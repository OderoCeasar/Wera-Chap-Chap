from django.utils.text import normalize_newlines
from rest_framework import serializers
from django.contrib.auth import authenticate
from .models import User, Device


class UserRegistrationSerializer(serializers.ModelSerializer):
    password = serializers.CharField(write_only=True, min_length=8)
    password_confirm = serializers.CharField(write_only=True)

    class Meta:
        model = User
        fields = ['id', 'username', 'email', 'phone', 'first_name', 'last_name', 'user_type', 'password', 'password_confirm']
        extra_kwargs = {
            'email': {'required': False},
            'first_name': {'required': False},
            'last_name': {'required': False},
        }

    def validat(self, data):
        if data.get('password') != data.get('password_confirm'):
            raise serializers.ValidationError('Passwords do not match')
        return data

    def create(self, validated_data):
        validated_data.pop('password_confirm')
        password = validated_data.pop('password')
        user = User.objects.create_user(password=password, **validated_data)
        return user

class UserLoginSerializer(serializers.Serializer):
    phone = serializers.CharField(required=False)
    username = serializers.CharField(required=False)
    password = serializers.CharField(write_only=True)

    def validate(self, data):
        phone = data.get('phone')
        username = data.get('username')
        password = data.get('password')

        if not phone and not username:
            raise serializers.ValidationError('Please provide either phone or username')

        user = None
        if phone:
            user = authenticate(phone=phone, password=password)
        elif username:
            try:
                user_obj = User.objects.get(username=username)
                user = authenticate(phone=user_obj.phone, password=password)
            except User.DoesNotExist:
                pass

            if not user:
                raise serializers.ValidationError('Invalid username or password')

            if not user.is_active:
                raise serializers.ValidationError('User account is disabled')

            data['user'] = user
            return data

class UserSerializer(serializers.ModelSerializer):
    full_name = serializers.CharField(source='get_full_name', read_only=True)

    class Meta:
        model = User
        fields = ['id', 'username', 'email', 'phone', 'first_name', 'last_name', 'full_name',
                  'user_type', 'location', 'profile_picture', 'is_verified', 'created_at', 'updated_at']
        read_only_fields = ['id', 'user_type', 'is_verified', 'created_at']


class UserUpdateSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = ['first_name', 'last_name', 'email', 'location', 'profile_picture']

class DeviceSerializer(serializers.ModelSerializer):
    class Meta:
        model = Device
        fields = ['id', 'device_id', 'fcm_token', 'device_type', 'is_active']
        read_only_fields = ['id']