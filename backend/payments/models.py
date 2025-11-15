from django.db import models
from django.conf import settings
from apps.tasks.models import Task


# Create your models here.
class Payment(models.Model):
    PAYMENT_METHOD_CHOICES = (
        ('mpesa', 'M-Pesa'),
        ('card', 'Card'),
        ('wallet', 'Wallet'),
    )

    STATUS_CHOICES = (
        ('pending', 'Pending'),
        ('successful', 'Successful'),
        ('failed', 'Failed'),
    )
    task = models.ForeignKey(Task, on_delete=models.CASCADE, related_name='payments', null=True, blank=True)
    user = models.ForeignKey(settings.AUTH_USER_MODEL, on_delete=models.CASCADE, related_name='payments')
    amount = models.DecimalField(max_digits=10, decimal_places=2)
    payment_method = models.CharField(choices=PAYMENT_METHOD_CHOICES, default='mpesa', max_length=10)

    #mpesa
    transaction_ref = models.CharField(max_length=255, unique=True)
    mpesa_receipt = models.TextField(blank=True)
    checkout_request_id = models.CharField(max_length=255, blank=True)
    merchant_request_id = models.CharField(max_length=255, blank=True)
    phone_number = models.CharField(max_length=15, blank=True)
    status = models.CharField(choices=STATUS_CHOICES, default='pending', max_length=20)
    result_code = models.CharField(max_length=20, blank=True)
    result_desc = models.TextField(blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = 'payments'
        indexes = [
            models.Index(fields=['transaction_ref']),
            models.Index(fields=['checkout_request_id']),
            models.Index(fields=['status']),
            models.Index(fields=['user']),
        ]
        ordering = ['-created_at']

    def __str__(self):
        return f"Payment {self.transaction_ref} - {self.status}"

    @property
    def is_successful(self):
        return self.status == 'successful'

class WithdrawalRequest(models.Model):
    STATUS_CHOICES = (
        ('pending', 'Pending'),
        ('processing', 'Processing'),
        ('completed', 'Completed'),
        ('failed', 'Failed'),
    )
    user = models.ForeignKey(settings.AUTH_USER_MODEL, on_delete=models.CASCADE, related_name='withdrawals')
    amount = models.DecimalField(max_digits=10, decimal_places=2)
    phone_number = models.CharField(max_length=15)
    status = models.CharField(choices=STATUS_CHOICES, default='pending', max_length=20)
    transaction_id = models.CharField(max_length=255, blank=True)
    result_desc = models.TextField(blank=True)
    processed_by = models.ForeignKey(settings.AUTH_USER_MODEL, on_delete=models.SET_NULL, null=True, blank=True, related_name='withdrawals_processed')
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    processed_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        db_table = 'withdrawals_requests'
        ordering = ['-created_at']

    def __str__(self):
        return f"Withdrawal {self.id} - {self.user.username} - {self.status}"
