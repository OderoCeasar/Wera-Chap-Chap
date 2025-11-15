from rest_framework import serializers
from .models import Task, TaskType, TaskApplication, UserService
from apps.users.serializers import UserSerializer

class TaskTypeSerializer(serializers.ModelSerializer):
    class Meta:
        model = TaskType
        fields = ['id', 'name', 'description', 'icon']

class TaskSerializer(serializers.ModelSerializer):
    posted_by = UserSerializer(read_only=True)
    assigned_to = UserSerializer(read_only=True)
    task_type_detail = TaskTypeSerializer(source='task_type', read_only=True)
    runner_payout = serializers.DecimalField(max_digits=10, decimal_places=2, read_only=True)

    class Meta:
        model = Task
        fields = [
            'id', 'title', 'description', 'task_type', 'task_type_detail',
            'posted_by', 'assigned_to', 'runner_payout', 'location', 'address',
            'latitude', 'longitude', 'budget', 'deadline', 'status',
            'total_paid', 'transaction_fee', 'proof_of_completion', 'completion_notes',
            'created_at', 'updated_at', 'completed_at',
        ]
        read_only_fields = [
            'id', 'posted_by', 'assigned_to', 'status', 'total_paid', 'transaction_fee',
            'created_at', 'updated_at', 'completed_at',
        ]

class TaskCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = Task
        fields = [
            'title', 'description', 'task_type', 'location',
            'address', 'latitude', 'longitude', 'budget', 'deadline',
        ]
        def validate_budget(self, value):
            if value < 100:
                raise serializers.ValidationError('Minimum budget is KES 100')
            return value

class TaskStatusUpdateSerializer(serializers.Serializer):
    status = serializers.ChoiceField(choices=['in_progress', 'completed'])
    proof = serializers.ImageField(required=False)
    notes = serializers.ImageField(required=False, allow_blank=True)

class TaskApplicationSerializer(serializers.ModelSerializer):
    runner = UserSerializer(read_only=True)
    task = TaskSerializer(read_only=True)

    class Meta:
        model = TaskApplication
        fields = ['id', 'task', 'runner', 'message', 'status', 'created_at']
        read_only_fields = ['id', 'status', 'created_at']

class TaskApproveRunnerSerializer(serializers.Serializer):
    runner_id = serializers.IntegerField()


class UserServiceSerializer(serializers.ModelSerializer):
    runner = UserSerializer(read_only=True)
    task_type_detail = TaskTypeSerializer(source='task_type', read_only=True)

    class Meta:
        model = UserService
        fields = [
            'id', 'runner', 'task_type', 'task_type_detail',
            'pricing', 'availability', 'description',
            'is_active', 'created_at'
        ]
        read_only_fields = ['id', 'created_at']

