from django.db import models
from django.contrib.auth.models import AbstractUser, BaseUserManager, PermissionsMixin
from django.core.validators import RegexValidator

# Create your models here.

class UserManager(BaseUserManager):
    def create_user(self, phone, password=None, **extra_fields):
        if not phone:
            raise ValueError('Phone number is required')
        user = self.model(phone=phone, **extra_fields)
        user.set_password(password)
        user.save(using=self._db)
        return user

    def create_superuser(self, phone, password, **extra_fields):
        extra_fields.setdefault('is_staff', True)
        extra_fields.setdefault('is_superuser', True)
        extra_fields.setdefault('user_type', 'admin')
        return self.create_user(phone, password, **extra_fields)

class User(AbstractUser, PermissionsMixin):
    USER_TYPE_CHOICES = (
        ('customer', 'Customer'),
        ('runner', 'Runner'),
        ('admin', 'Admin'),
    )


    phone_regex = RegexValidator(
        regex=r'^\+?254\d{9}$',
        message="Phone number must be in format +254703382631"
    )

    username = models.CharField(max_length=150, unique=True)
    email = models.EmailField(max_length=254, unique=True, null=True, blank=True)
    phone = models.CharField(max_length=10, unique=True, validators=[phone_regex])
    first_name = models.CharField(max_length=50, blank=True)
    last_name = models.CharField(max_length=50, blank=True)
    user_type = models.CharField(max_length=10, choices=USER_TYPE_CHOICES, default='customer')
    is_staff = models.BooleanField(default=False)
    is_active = models.BooleanField(default=True)
    is_verified = models.BooleanField(default=False)
    location = models.CharField(max_length=255, blank=True)
    profile_picture = models.ImageField(upload_to='profiles/', null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    objects = UserManager()
    USERNAME_FIELD = 'phone'
    REQUIRED_FIELDS = ['username']

    class Meta:
        db_table = 'users'
        indexes = [
            models.Index(fields=['phone']),
            models.Index(fields=['email']),
            models.Index(fields=['user_type']),
        ]

    def __str__(self):
        return f"{self.username} {self.phone}"

    def get_full_name(self):
        return f"{self.first_name} {self.last_name}".strip() or self.username

    @property
    def is_customer(self):
        return self.user_type == 'customer'

    @property
    def is_runner(self):
        return self.user_type == 'runner'

# Device //stores token for push notifications
class Device(models.Model):
    user = models.ForeignKey(User, on_delete=models.CASCADE, related_name='devices')
    device_id = models.CharField(max_length=255, unique=True)
    fcm_token = models.TextField()
    device_type = models.CharField(max_length=20, choices=[
        ('ios', 'iOS'),
        ('android', 'Android'),
        ('web', 'Web'),
    ])
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = 'devices'

    def __str__(self):
        return f"{self.user.username} - {self.device_type}"

# AuditLog // user actions for compliance
class AuditLog(models.Model):
    user = models.ForeignKey(User, on_delete=models.SET_NULL, null=True, related_name='audit_logs')
    action = models.CharField(max_length=100)
    resource_type = models.CharField(max_length=50)
    resource_id = models.BigIntegerField(null=True)
    details = models.JSONField(null=True, blank=True)
    ip_address = models.GenericIPAddressField(null=True)
    user_agent = models.TextField(blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = 'audit_logs'
        indexes = [
            models.Index(fields=['user', 'created_at']),
            models.Index(fields=['resource_type', 'resource_id']),
        ]

    def __str__(self):
        return f"{self.user.user} - {self.action} at {self.created_at}"

