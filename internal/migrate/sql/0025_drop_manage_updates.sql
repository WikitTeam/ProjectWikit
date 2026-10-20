-- compat: compatible
DELETE FROM web_role_permissions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM web_role_restrictions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM web_rolepermissionsoverride_permissions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM web_rolepermissionsoverride_restrictions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM auth_group_permissions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM web_user_user_permissions WHERE permission_id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
DELETE FROM auth_permission WHERE id IN
    (SELECT p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = 'manage_updates');
