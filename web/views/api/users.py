from django.http import HttpRequest

from shared_data import shared_users
from . import APIView, APIError, takes_json

from web.models import ActionLogEntry
from django.utils.http import urlsafe_base64_encode  
from django.utils.encoding import force_bytes  
from web.views.invite import account_activation_token  
from web.models.site import get_current_site  
from django.contrib.auth import get_user_model  
  
User = get_user_model()  
  
class GenerateInviteLinkView(APIView):  
    @takes_json  
    def post(self, request: HttpRequest):  
        if not request.user.has_perm('roles.manage_users'):  
            raise APIError('权限不足', 403)  
          
        email = self.json_input.get('email')  
        if not email:  
            raise APIError('未指定邮箱', 400)  
          
        roles = self.json_input.get('roles', [])  
          
        try:  
            user, created = User.objects.get_or_create(email=email)  
            if not created:  
                raise APIError('该邮箱的用户已存在', 409)  
              
            if roles:  
                user.roles.set(roles)  
            user.is_active = False  
            user.username = f'user-{user.id}'  
            user.save()  
               
            token = account_activation_token.make_token(user)  
            uid = urlsafe_base64_encode(force_bytes(user.pk))  
            site = get_current_site()  
            invitation_url = f"{request.scheme}://{site.domain}/-/accept/{uid}/{token}"  
              
            return self.render_json(200, {  
                'email': email,  
                'invitationUrl': invitation_url,  
                'userId': user.id  
            })  
              
        except Exception as e:  
            raise APIError('创建邀请时出错', 500)

class AllUsersView(APIView):
    def get(self, request: HttpRequest):
        return self.render_json(200, shared_users.get_all_users())


class AdminSusActivityApiView(APIView):
    def get(self, request: HttpRequest):
        if not request.user.has_perm('roles.view_sensitive_info'):
            raise APIError('权限不足', 403)
        items = list()
        for logentry in ActionLogEntry.objects.prefetch_related('user').distinct('user', 'origin_ip'):
            items.append({
                'user': {
                    'id': logentry.user.id,
                    'name': logentry.user.username,
                },
                'ip': logentry.origin_ip
            })
        return self.render_json(200, items)
