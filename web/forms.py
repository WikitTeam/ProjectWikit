from django import forms
from django.core.validators import RegexValidator

from web.models.roles import Role
from web.models.users import User


class UserProfileForm(forms.ModelForm):  
    full_name = forms.CharField(label='姓名', required=False)  
      
    class Meta:  
        model = User  
        fields = ['full_name', 'bio', 'avatar']   
      
    def __init__(self, *args, **kwargs):  
        super().__init__(*args, **kwargs)  
 
        if self.instance:  
            if self.instance.first_name and self.instance.last_name:  
                self.fields['full_name'].initial = f"{self.instance.first_name} {self.instance.last_name}"  
            elif self.instance.first_name:  
                self.fields['full_name'].initial = self.instance.first_name  
      
    def save(self, commit=True):  
        full_name = self.cleaned_data.get('full_name', '').strip()  
        if full_name:  
            parts = full_name.rsplit(' ', 1)  
            self.instance.first_name = parts[0]  
            self.instance.last_name = parts[1] if len(parts) > 1 else ''  
        else:  
            self.instance.first_name = ''  
            self.instance.last_name = ''  
        return super().save(commit=commit)

class InviteForm(forms.Form):
    _selected_user = forms.IntegerField(widget=forms.MultipleHiddenInput, required=False)
    email = forms.EmailField(widget=forms.EmailInput(attrs={'class': 'vTextField'}))
    roles = forms.ModelMultipleChoiceField(label='用户组权限', queryset=Role.objects.exclude(slug__in=['everyone', 'registered']), required=False)


class CreateAccountForm(forms.Form):
    username = forms.CharField(label='用户名', required=True, validators=[RegexValidator(r'^[A-Za-z0-9_-]+$', '用户名格式无效')])
    password = forms.CharField(label='密码', widget=forms.PasswordInput(), required=True)
    password2 = forms.CharField(label='确认密码', widget=forms.PasswordInput(), required=True)

    def clean_password2(self):
        cd = self.cleaned_data
        if cd['password'] != cd['password2']:
            raise forms.ValidationError('密码不匹配')
        return cd['password2']


class CreateBotForm(forms.Form):
    username = forms.CharField(
        label='机器人昵称',
        required=True,
        validators=[
                RegexValidator(r'^[A-Za-z0-9_-]+$', '用户名格式无效')
            ]
        )