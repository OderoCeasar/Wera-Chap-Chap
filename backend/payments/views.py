from rest_framework import viewsets, status, permissions
from rest_framework.decorators import api_view, permission_classes, action
from rest_framework.responnse import Response
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator
from django.shortcuts import get_object_or_404
from django.db import transaction
from decimal import Decimal
import logging
import uuid

from .models import Payment, WithdrawalRequest
from.serializers import PaymentSerializer, WithdrawalRequestSerializer
from apps.tasks.models import Task
from apps.wallets.models import Wallet
from utils.mpesa import mpesa_client
from .tasks import process_mpesa_callback

logger = logging.getLogger(__name__)

# Create your views here.
class PaymentViewSet(viewsets.ReadOnlyModelViewSet):
    serializer_class = PaymentSerializer
    permission_classes = [permissions.IsAuthenticated]

    def get_queryset(self):
        return Payment.objects.filter(user=self.request.user).select_related('task')


class InitiatePaymentViewSet(viewsets.ViewSet):
    permission_classes = [permissions.IsAuthenticated]

    @action(detail=True, methods=['post'], url_path='pay')
    def initiate_payment(self, request, pk=None):
        # initiate mpesa stk push
        task  = get_object_or_404(Task, pk=pk)

        if task.posted_by != request.user:
            return Response(
                {'error': 'Task payment has already been processed.'},
                status=status.HTTP_400_BAD_REQUEST
            )

        phone = request.data.get('phone', request.user.phone)
        amount = Decimal(request.data.get('amount', task.budget))

        if amount < task.budget:
            return Response(
                {'error': f'Amount must be at least KES {task.budget}'},
                status=status.HTTP_400_BAD_REQUEST
            )
        transaction_fee = amount * Decimal('0.10')
        transaction_ref = f"WERA{uuid.uuid4().hex[:10].upper()}"

        try:
            #create payment record
            payment = Payment.objects.create(
                task=task,
                user=request.user,
                amount=amount,
                payment_method='mpesa',
                transaction_ref=transaction_ref,
                phone_number=phone,
                status='pending',
            )

            #initiate STK push
            result = mpesa_client.stk_push(
                phone_number=phone,
                amount=amount,
                account_reference=f"Task-{task.id}",
                transaction_desc=f"Payment for {task.title}"
            )

            if result['success']:
                data = result['data']

                #update payment with Mpesa details
                payment.checkout_request_id = data.get('CheckoutRequestID', '')
                payment.merchant_request_id = data.get('MerchantRequestID', '')
                payment.save()

                #update task with fee
                task.transaction_fee = transaction_fee
                task.save()

                return Response({
                    'message': 'STK Push initiated successfully',
                    'checkout_request_id': payment.checkout_request_id,
                    'payment_id': payment.id,
                    'response_code': data.get('ResponseCode'),
                    'response_description': data.get('ResponseDescription'),
                }, status=status.HTTP_200_OK)
            else:
                payment.status = 'failed'
                payment.result_desc = result.get('error', 'Failed to initiate payment')
                payment.save()

                return Response({
                    'error': 'Failed to initiate payment',
                    'details': result.get('error'),
                }, status=status.HTTP_400_BAD_REQUEST)
        except Exception as e:
            logger.error(f"Payment initiation error: {e}")
            return Response({
                'error': 'An error occurred while processing payment',
            }, status=status.HTTP_500_INTERNAL_SERVER_ERROR)

@csrf_exempt
@api_view(['POST'])
@permission_classes([permissions.AllowAny])
def mpesa_callback(request):
    #mpesa cofirmation from safaricom
    try:
        logger.info(f"M-Pesa callback received: {request.data}")
        process_mpesa_callback.delay(request.data)

        return Response({
            'ResultCode': 0,
            'ResultDesc': 'Accepted',
        }, status=status.HTTP_200_OK)
    except Exception as e:
        logger.error(f"M-Pesa callback error: {e}")
        return Response({
            'ResultCode': 1,
            'ResultDesc': 'Failed',
        }, status=status.HTTP_200_OK)

@csrf_exempt
@api_view(['POST'])
@permission_classes([permissions.AllowAny])
def mpesa_timeout(request):
    # handle mpesa timeout callbacks
    logger.warning(f"M-Pesa timeout received: {request.data}")
    return Response({'ResultCode': 0, 'ResultDesc': 'Accepted'})

@csrf_exempt
@api_view(['POST'])
@permission_classes([permissions.AllowAny])
def mpesa_b2c_callback(request):
    # handle withdrawal callbacks
    logger.info(f"M-Pesa B2C callback: {request.data}")

    try:
        result = request.data.get('Result', {})
        result_code = result.get('ResultCode')
        conversation_id = result.get('ConversationId')

        # withdrawal request
        try:
            withdrawal = WithdrawalRequest.objects.get(transaction_id=conversation_id)

            if result_code == 0:
                withdrawal.status = 'completed'
                withdrawal.result_desc = result.get('ResultDesc', 'Success')
            else:
                withdrawal.status = 'failed'
                withdrawal.result_desc = result.get('ResultDesc', 'Failed')
            withdrawal.save()
        except WithdrawalRequest.DoesNotExist:
            logger.warning(f"Withdrawal not found for conversation: {conversation_id}")

        return Response({'ResultCode': 0, 'ResultDesc': 'Accepted'})
    except Exception as e:
        logger.error(f"B2C callback error: {e}")
        return Response({'ResultCode': 1, 'ResultDesc': 'Failed'})

