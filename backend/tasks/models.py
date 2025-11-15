from django.db import models
from django.conf import settings
from django.core.validators import MinValueValidator


# Create your models here.
# Task types
class TaskType(models.Model):
    name = models.CharField(max_length=100, unique=True)
    description = models.TextField(blank=True)
    icon = models.CharField(max_length=50, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = 'task_types'

    def __str__(self):
        return self.name


# Tasks
class Task(models.Model):
    STATUS_CHOICES = (
        ('payment_pending', 'Payment Pending'),
        ('new', 'New'),
        ('assigned', 'Assigned'),
        ('in_progress', 'In Progress'),
        ('completed', 'Completed'),
        ('cancelled', 'Cancelled'),
    )


    title = models.CharField(max_length=100)
    description = models.TextField()
    task_type = models.ForeignKey(TaskType, on_delete=models.RESTRICT, related_name='tasks')
    posted_by = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name='posted_tasks',
    )
    assigned_to = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        related_name='assigned_tasks',
    )
    location = models.CharField(max_length=255)
    address = models.CharField(max_length=512, blank=True)
    latitude = models.DecimalField(max_digits=9, decimal_places=6, null=True, blank=True)
    longitude = models.DecimalField(max_digits=9, decimal_places=6, null=True, blank=True)

    budget = models.DecimalField(max_digits=10, decimal_places=2, validators=[MinValueValidator(0)])
    deadline = models.DateField(null=True, blank=True)
    status = models.CharField(max_length=20, choices=STATUS_CHOICES, default='payment_pending')
    total_paid = models.DecimalField(max_digits=10, decimal_places=2, default=0)
    transaction_fee = models.DecimalField(max_digits=10, decimal_places=2, default=0)
    proof_of_completion = models.ImageField(upload_to='tasks_proofs/', null=True, blank=True)
    completion_notes = models.TextField(blank=True)

    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    completed_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        db_table = 'tasks'
        indexes = [
            models.Index(fields=['posted_by']),
            models.Index(fields=['assigned_to']),
            models.Index(fields=['status']),
            models.Index(fields=['created_at']),
            models.Index(fields=['location']),
        ]
        ordering = ['-created_at']

    def __str__(self):
        return f"{self.title} - {self.status}"

    @property
    def runner_payout(self):
        # calculating the runner payout
        return self.total_paid - self.transaction_fee

    def can_be_assigned_to(self, user):
        # check is a task can be assigned
        return(
            user.is_runner and
            self.status in ['new', 'payment_pending'] and
            self.assigned_to is None
        )

class TaskApplication(models.Model):
    task = models.ForeignKey(Task, on_delete=models.CASCADE, related_name='applications')
    runner = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name='task_applications',
    )
    message = models.TextField(blank=True)
    status = models.CharField(
        max_length=20,
        choices=[
            ('pending', 'Pending'),
            ('accepted', 'Accepted'),
            ('rejected', 'Rejected'),
        ],
        default='pending',

    )
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = 'task_applications'
        unique_together = (('task', 'runner'),)
        ordering = ['-created_at']

    def __str__(self):
        return f"{self.runner.username} -> {self.task.title}"


class USerService(models.Model):
    runner = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name='offered_services',
    )
    task_type = models.ForeignKey(TaskType, on_delete=models.CASCADE)
    pricing = models.DecimalField(max_digits=10, decimal_places=2, validators=[MinValueValidator(0)])
    availability = models.CharField(max_length=255, blank=True)
    description = models.TextField(blank=True)
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = 'user_services'
        unique_together = (('runner', 'task_type'),)

    def __str__(self):
        return f"{self.runner.username} -> {self.task_type.name}"