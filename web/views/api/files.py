from django.conf import settings
from django.db.models import Sum
from django.http import HttpRequest
import os.path
from uuid import uuid4

from renderer.utils import render_user_to_json
from . import APIView, APIError, takes_json

from web.controllers import articles
import urllib.parse

from ...models.files import File


class FileView(APIView):
    @staticmethod
    def _validate_request(request: HttpRequest, article_name_or_article, edit=True):
        article = articles.get_article(article_name_or_article)
        if article is None:
            category = articles.get_article_category(article_name_or_article)
            if not request.user.has_perm('roles.view_articles', category):
                raise APIError('权限不足', 403)
            raise APIError('页面未找到', 404)
        if edit and not request.user.has_perm('roles.manage_article_files', article):
            raise APIError('权限不足', 403)
        return article


class GetOrUploadView(FileView):
    def get(self, request: HttpRequest, article_name):
        category = articles.get_article_category(article_name)
        if not request.user.has_perm('roles.view_articles', category):
            raise APIError('权限不足', 403)
        article = self._validate_request(request, article_name, edit=False)
        files = articles.get_files_in_article(article)
        output = []
        current_files_size, absolute_files_size = articles.get_file_space_usage()
        for file in files:
            output.append({'id': file.id, 'name': file.name, 'size': file.size, 'createdAt': file.created_at, 'author': render_user_to_json(file.author), 'mimeType': file.mime_type})
        data = {
            'pageId': article.full_name,
            'files': output,
            'softLimit': settings.MEDIA_UPLOAD_LIMIT,
            'hardLimit': settings.ABSOLUTE_MEDIA_UPLOAD_LIMIT,
            'softUsed': current_files_size,
            'hardUsed': absolute_files_size
        }
        return self.render_json(200, data)

    def post(self, request: HttpRequest, article_name):
        article = self._validate_request(request, article_name)
        file_name = request.headers.get('x-file-name')
        if not file_name:
            raise APIError('缺少文件名', 400)
        file_name = urllib.parse.unquote(file_name)
        existing_file = articles.get_file_in_article(article, file_name)
        if existing_file:
            raise APIError('同名文件已存在', 409)
        _, ext = os.path.splitext(file_name)
        media_name = str(uuid4()) + ext
        new_file = File(name=file_name, media_name=media_name, author=request.user, article=article)
        local_media_dir = os.path.dirname(new_file.local_media_path)
        if not os.path.exists(local_media_dir):
            os.makedirs(local_media_dir, exist_ok=True)
        # 上传文件到临时存储
        current_files_size, absolute_files_size = articles.get_file_space_usage()
        try:
            size = 0
            with open(new_file.local_media_path, 'wb') as f:
                while True:
                    chunk = request.read(102400)
                    size += len(chunk)
                    # 处理文件大小限制
                    if (settings.MEDIA_UPLOAD_LIMIT > 0 and current_files_size + size > settings.MEDIA_UPLOAD_LIMIT) or \
                            (settings.ABSOLUTE_MEDIA_UPLOAD_LIMIT > 0 and absolute_files_size + size > settings.ABSOLUTE_MEDIA_UPLOAD_LIMIT):
                        raise APIError('超过文件上传限制', 413)
                    if not chunk:
                        break
                    f.write(chunk)
            new_file.size = size
            new_file.mime_type = request.headers.get('content-type', 'application/octet-stream')
            articles.add_file_to_article(article, new_file, user=request.user)
        except:
            if os.path.exists(new_file.local_media_path):
                os.unlink(new_file.local_media_path)
            raise
        return self.render_json(200, {'status': 'ok'})


class RenameOrDeleteView(FileView):
    @staticmethod
    def _get_file_and_article(file_id):
        file = File.objects.get(id=file_id)
        if file is None:
            return None, None
        return file.article, file

    def delete(self, request: HttpRequest, file_id):
        article, file = self._get_file_and_article(file_id)
        article = self._validate_request(request, article)
        if file is None:
            raise APIError('文件不存在', 404)
        articles.delete_file_from_article(article, file, user=request.user)
        return self.render_json(200, {'status': 'ok'})

    @takes_json
    def put(self, request: HttpRequest, file_id):
        article, file = self._get_file_and_article(file_id)
        article = self._validate_request(request, article)
        if file is None:
            raise APIError('文件不存在', 404)
        data = self.json_input
        if not isinstance(data, dict) or 'name' not in data:
            raise APIError('无效请求', 400)
        if not data['name']:
            raise APIError('缺少文件名', 400)
        existing_file = articles.get_file_in_article(article, data['name'])
        if existing_file and existing_file.id != file.id:
            raise APIError('同名文件已存在', 409)
        articles.rename_file_in_article(article, file, data['name'], user=request.user)
        return self.render_json(200, {'status': 'ok'})