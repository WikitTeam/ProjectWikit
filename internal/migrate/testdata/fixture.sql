INSERT INTO public.web_user VALUES (1, 'pbkdf2_sha256$1000000$BCEp0IYLIrpb$UDF4z3zPqN7ji9u9HfJJOCtlGJmOFWT/cCF4CZ+KjWc=', '2026-09-05 11:58:21.332836+00', false, '', '', '', '2026-08-20 07:26:18.01709+00', 'seeduser', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (30, '', NULL, false, '', '', '', '2026-08-24 10:49:07.504795+00', 'probe-author', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, 'Probe Author', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (32, '', NULL, false, '', '', '', '2026-08-24 10:49:07.53524+00', 'probevoter', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (188, '', NULL, true, '', '', '', '2026-08-30 04:50:27.310595+00', 'probe-staff', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, 'Probe Staff', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (33, '', NULL, false, '', '', '', '2026-08-24 10:50:52.619683+00', 'probecrowd0', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (34, '', NULL, false, '', '', '', '2026-08-24 10:50:52.624817+00', 'probecrowd1', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (35, '', NULL, false, '', '', '', '2026-08-24 10:50:52.628825+00', 'probecrowd2', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (36, '', NULL, false, '', '', '', '2026-08-24 10:50:52.63332+00', 'probecrowd3', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (67, 'pbkdf2_sha256$1000000$gC5OHX7CJ3MYOm0cuAXtuQ$huWdajOZn3G2GeUITCP55CMR/PAl+x3RUbmAg3ql5Eg=', '2026-08-26 16:57:22.975846+00', true, '', '', '', '2026-08-26 16:51:54.56825+00', 'wikitadmin', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, 'Wikit Admin', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (37, '', NULL, false, '', '', '', '2026-08-24 10:50:52.637053+00', 'probecrowd4', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (38, '', NULL, false, '', '', '', '2026-08-24 10:50:52.64114+00', 'probecrowd5', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (39, '', NULL, false, '', '', '', '2026-08-24 10:50:52.644782+00', 'probecrowd6', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (40, '', NULL, false, '', '', '', '2026-08-24 10:50:52.648893+00', 'probecrowd7', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, NULL, NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (189, 'pbkdf2_sha256$1000000$HmjrOdsw0zJGkixCtLijqe$T0mZ6boolXY+E0oU0XWLwl0uUwYqzEY171rjmmrnDhg=', '2026-08-31 11:10:07.711335+00', false, '', '', '', '2026-08-31 11:09:57.662033+00', 'demo', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, '演示账号', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (190, 'pbkdf2_sha256$1000000$JMENisbO7lsFXsMDdPLPhT$Qd8yTZEHijgfGE4w1HuUoVcyUTxRIVDfuwfRQhcRSXs=', '2026-08-31 11:10:59.185714+00', true, '', '', '', '2026-08-31 11:10:47.747027+00', 'demoadmin', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, '演示管理员', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (191, 'pbkdf2_sha256$1000000$4RwXBOBgSKGZytCSN7NIEV$ZTn+JVD/MzSF1XuCPg+tOLVTOEsCSUyaI4teF1r5swg=', '2026-08-31 12:07:17.690408+00', false, '', '', 'newcomer@example.org', '2026-08-31 12:07:05.279873+00', 'newcomer', NULL, 'normal', '', '', NULL, true, NULL, true, NULL, true, 'Newcomer', NULL, '', '', NULL, NULL);
INSERT INTO public.web_user VALUES (31, '', '2026-08-31 12:02:19.152912+00', false, '', '', '', '2026-08-24 10:49:07.531608+00', '576c0df3-8a28-4468-9770-ede851d88c67', 'probe-wd-original', 'wikidot', '', '', NULL, true, NULL, false, NULL, true, 'Probe WD', NULL, '', '', NULL, NULL);

INSERT INTO public.dynamic_preferences_users_userpreferencemodel VALUES (28, 'qol', 'advanced_source_editor_enabled', 'False', 67);
INSERT INTO public.dynamic_preferences_users_userpreferencemodel VALUES (29, 'qol', 'advanced_source_editor_enabled', 'False', 188);
INSERT INTO public.dynamic_preferences_users_userpreferencemodel VALUES (27, 'qol', 'advanced_source_editor_enabled', 'True', 30);
INSERT INTO public.dynamic_preferences_users_userpreferencemodel VALUES (30, 'qol', 'advanced_source_editor_enabled', 'False', 1);

INSERT INTO public.web_actionlogentry VALUES (1, 'wikitadmin', 'vote', '{"is_new": true, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 08:43:17.773435+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (2, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": -1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:43:19.781634+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (3, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": -1, "old_vote": -1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:44:32.149287+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (4, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": -1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:44:33.641333+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (5, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": null, "old_vote": 1.0, "is_change": false, "is_remove": true}', '2026-08-30 08:44:35.249927+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (6, 'wikitadmin', 'vote', '{"is_new": true, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 08:44:43.063729+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (7, 'probe-staff', 'vote', '{"is_new": true, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 08:48:03.320948+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (8, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:03.532547+00', '172.18.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (9, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:13.316658+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (10, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:13.475767+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (11, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:13.676248+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (12, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:13.839397+00', '172.18.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (13, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:14.012926+00', '172.18.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (14, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:48:14.205534+00', '172.18.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (15, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:49:01.979231+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (16, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:49:02.170022+00', '172.18.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (17, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:49:19.951337+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (18, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:54:30.145355+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (19, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": -1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 08:54:31.71078+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (20, 'probe-staff', 'vote', '{"is_new": true, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 08:57:46.832836+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (21, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": -1, "old_vote": -1.0, "is_change": true, "is_remove": false}', '2026-08-30 09:02:36.418238+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (22, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": -1.0, "is_change": true, "is_remove": false}', '2026-08-30 09:02:37.714237+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (23, 'probe-staff', 'vote', '{"is_new": true, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 09:04:03.718509+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (24, 'probe-staff', 'vote', '{"is_new": false, "article": "Probe Full (probe:full)", "new_vote": -1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 09:05:01.313351+00', '127.0.0.1', 188);
INSERT INTO public.web_actionlogentry VALUES (25, 'wikitadmin', 'vote', '{"is_new": false, "article": "scp-173 (scp-173)", "new_vote": 1, "old_vote": 1.0, "is_change": true, "is_remove": false}', '2026-08-30 09:09:18.004374+00', '127.0.0.1', 67);
INSERT INTO public.web_actionlogentry VALUES (26, 'probe-staff', 'vote', '{"is_new": true, "article": "Probe Full (probe:full)", "new_vote": 1, "old_vote": null, "is_change": false, "is_remove": false}', '2026-08-30 09:11:45.055907+00', '127.0.0.1', 188);

INSERT INTO public.web_article VALUES (122, 'forum', 'category', 'category', false, '2026-08-26 16:12:44.792185+00', '2026-08-26 16:12:44.79952+00', NULL, '20db24df-6901-432a-8502-43629c9c5dd6', DEFAULT);
INSERT INTO public.web_article VALUES (137, 'probestars', 'quarter', 'Probe Quarter', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'b617d328-a33d-488c-9fb3-69159fb24987', DEFAULT);
INSERT INTO public.web_article VALUES (126, 'probe', 'bydisplay', 'Probe By Display Name', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '909df4ed-d878-4d88-8464-171165e882b2', DEFAULT);
INSERT INTO public.web_article VALUES (132, 'probe', 'listempty', 'Probe List Empty', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '115e44f6-c545-48d2-9338-e0d5441722ab', DEFAULT);
INSERT INTO public.web_article VALUES (3, 'component', 'box', 'box', false, '2026-08-20 07:26:18.337302+00', '2026-08-20 07:26:18.347679+00', NULL, 'c0586cb9-3bd4-4fb9-bfa6-e809d77a1328', DEFAULT);
INSERT INTO public.web_article VALUES (115, 'probe', 'tagged', 'Probe Tagged', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'f351f8be-ecfa-4410-b848-36bc36ff1cd5', DEFAULT);
INSERT INTO public.web_article VALUES (129, 'probe', 'listnowrap', 'Probe List No Wrapper', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '2180212d-66f7-47d7-9c72-f5af9b1955c1', DEFAULT);
INSERT INTO public.web_article VALUES (123, 'forum', 'thread', 'thread', false, '2026-08-26 16:12:44.806389+00', '2026-08-26 16:12:44.812809+00', NULL, 'bdcc7125-f563-4d7d-86a1-ed984a3cdf2e', DEFAULT);
INSERT INTO public.web_article VALUES (5, '_default', 'scp-173', 'scp-173', false, '2026-08-20 07:26:18.36665+00', '2026-08-20 07:26:18.377982+00', NULL, '556e83fe-c554-4b37-b9f0-0c5044c45dc3', DEFAULT);
INSERT INTO public.web_article VALUES (8, 'component', 'probe-var', 'Shared Component', false, '2026-08-23 08:47:08.282178+00', '2026-08-23 08:47:08.332887+00', NULL, '635e0386-4862-4a0d-b7bd-c1cfb31d40d1', DEFAULT);
INSERT INTO public.web_article VALUES (17, 'probe', 'host', 'Probe Host', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '004ee6a2-4e0d-42a2-ab46-ae837a955c7f', DEFAULT);
INSERT INTO public.web_article VALUES (112, 'probe', 'redirect', 'Probe Redirect', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '574d5efc-84c0-46ec-9a21-52eeed32f148', DEFAULT);
INSERT INTO public.web_article VALUES (9, '_default', 'probe-a', 'Page A', false, '2026-08-23 08:47:08.337019+00', '2026-08-23 08:47:08.351223+00', NULL, 'a1d3ec2a-62b8-4f9e-8f3e-d6f1945e17b8', DEFAULT);
INSERT INTO public.web_article VALUES (10, '_default', 'probe-b', 'Page B', false, '2026-08-23 08:47:08.356311+00', '2026-08-23 08:47:08.372248+00', NULL, 'a674ce40-c859-4d9e-a77b-33a7fb48df28', DEFAULT);
INSERT INTO public.web_article VALUES (116, 'probe', 'taggedplain', 'Probe Tagged Plain', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'd904d112-0552-4414-ab0d-a56f9ec71e80', DEFAULT);
INSERT INTO public.web_article VALUES (124, 'forum', 'start', 'start', false, '2026-08-26 16:12:44.819086+00', '2026-08-26 16:12:44.826374+00', NULL, '6316c01c-dca1-47d6-b6ae-67c7b5c4a70a', DEFAULT);
INSERT INTO public.web_article VALUES (138, 'probeoff', 'unratable', 'Probe Unratable', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'b5ae2ba0-3975-4a01-bd18-50422d7ca8d7', DEFAULT);
INSERT INTO public.web_article VALUES (125, 'nav', 'top-impl', 'top-impl', false, '2026-08-26 16:12:44.834388+00', '2026-08-26 16:12:44.840541+00', NULL, 'fda33fce-d131-4887-af95-4c57bdbb708d', DEFAULT);
INSERT INTO public.web_article VALUES (1, 'nav', 'top', 'top', false, '2026-08-20 07:26:18.286289+00', '2026-08-26 16:12:44.852216+00', NULL, 'cfb976b6-df07-4de5-99fa-066f8d95ca42', DEFAULT);
INSERT INTO public.web_article VALUES (2, 'nav', 'side', 'side', false, '2026-08-20 07:26:18.322901+00', '2026-08-26 16:12:44.865597+00', NULL, '8b3d0619-586c-4825-bf86-e8f438ed2bb8', DEFAULT);
INSERT INTO public.web_article VALUES (198, 'probe', 'changes', 'Probe Changes', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '3bfc95cd-ab75-4746-9c20-25c38fe2ae40', DEFAULT);
INSERT INTO public.web_article VALUES (130, 'probe', 'listsections', 'Probe List Sections', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '571c58eb-32a3-4986-ba4a-2f7f4fe115b9', DEFAULT);
INSERT INTO public.web_article VALUES (131, 'probe', 'listtags', 'Probe List Tags', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'abd8cb8d-59f2-45b6-8bcb-18ca01b84cbf', DEFAULT);
INSERT INTO public.web_article VALUES (127, 'probe', 'listed', 'Probe Listed', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '402708c3-5e3b-449d-b96b-3d1392e95539', DEFAULT);
INSERT INTO public.web_article VALUES (13, 'probe', 'bare', 'Probe Bare', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '9b5dadcb-bb25-468b-880d-b85734c851f3', DEFAULT);
INSERT INTO public.web_article VALUES (196, 'probecss', 'styled', 'Probe Styled', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '12998070-4341-4680-a491-36332e6c0125', DEFAULT);
INSERT INTO public.web_article VALUES (113, 'probe', 'described', 'Probe Described', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'cebb5408-4257-403a-854f-86203d16a298', DEFAULT);
INSERT INTO public.web_article VALUES (14, 'probestars', 'rated', 'Probe Rated', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '5f2e2258-312b-44a1-bd74-ecb54fdb4476', DEFAULT);
INSERT INTO public.web_article VALUES (4, '_default', 'main', 'main', false, '2026-08-20 07:26:18.351937+00', '2026-08-26 16:12:44.642178+00', NULL, 'ff3f591b-cb42-491f-9910-057f16724c1a', DEFAULT);
INSERT INTO public.web_article VALUES (114, 'probe', 'imaged', 'Probe Imaged', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '2224ce44-cd57-4cea-91a5-c4171df32a5c', DEFAULT);
INSERT INTO public.web_article VALUES (118, '_default', 'wiki-syntax-guide', 'wiki-syntax-guide', false, '2026-08-26 16:12:44.654543+00', '2026-08-26 16:12:44.741391+00', NULL, '295a3d02-9f5e-45ca-81b8-fdecbcdaa720', DEFAULT);
INSERT INTO public.web_article VALUES (135, 'probestars', 'unrated', 'Probe Unrated', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '00cc8b22-b29f-4fea-a797-d4fc233a7f02', DEFAULT);
INSERT INTO public.web_article VALUES (119, 'search', 'site', 'site', false, '2026-08-26 16:12:44.750837+00', '2026-08-26 16:12:44.758612+00', NULL, '526a25af-efd8-4c24-a51a-64b51201dd81', DEFAULT);
INSERT INTO public.web_article VALUES (197, 'probecss', 'styledhead', 'Probe Styled Head', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '8a0e9fbf-0de5-444e-9200-b344d9eee65a', DEFAULT);
INSERT INTO public.web_article VALUES (120, 'forum', 'recent-posts', 'recent-posts', false, '2026-08-26 16:12:44.766694+00', '2026-08-26 16:12:44.773398+00', NULL, 'a51fe677-0008-4c83-87cf-16013715a98f', DEFAULT);
INSERT INTO public.web_article VALUES (128, 'probe', 'listjoined', 'Probe List Joined', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '8942f67c-bac1-4610-ba13-db758653396e', DEFAULT);
INSERT INTO public.web_article VALUES (121, 'forum', 'new-thread', 'new-thread', false, '2026-08-26 16:12:44.779666+00', '2026-08-26 16:12:44.785956+00', NULL, '96b18252-c2a2-4595-8e97-ac86a9501944', DEFAULT);
INSERT INTO public.web_article VALUES (133, 'probe', 'listurl', 'Probe List Url', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'be874be4-54d6-4b12-971f-eccd6ee05f42', DEFAULT);
INSERT INTO public.web_article VALUES (139, 'probestars', 'third', 'Probe Third', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'b835e572-5c8f-498c-ab19-1cdc8481904d', DEFAULT);
INSERT INTO public.web_article VALUES (117, 'probe', 'unknownmodule', 'Probe Unknown Module', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '78663848-ceed-475b-a373-b756af10f036', DEFAULT);
INSERT INTO public.web_article VALUES (15, 'probe', 'half', 'Probe Half', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, 'a0a9e44d-06d1-444c-a6f1-9e9666d009ba', DEFAULT);
INSERT INTO public.web_article VALUES (134, 'probe', 'listbyvotes', 'Probe List By Votes', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '79db7c53-3cf8-4100-a803-7bd4a2ac4f53', DEFAULT);
INSERT INTO public.web_article VALUES (12, 'probe', 'full', 'Probe Full', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', 11, '0f69121c-4673-408a-9441-7c03b40af8fa', DEFAULT);
INSERT INTO public.web_article VALUES (16, 'probe', 'included', 'Included Page', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '7f1c329b-cfd6-49d8-8284-5c84770022df', DEFAULT);
INSERT INTO public.web_article VALUES (11, 'probe', 'parent', 'Probe Parent', false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', NULL, '70d39d77-c55a-4930-abd5-e3802d5a1ced', DEFAULT);
INSERT INTO public.web_article VALUES (205, '_default', 'pwikit-demo', 'pwikit 新功能演示', false, '2026-08-31 10:59:02.384683+00', '2026-08-31 10:59:02.384683+00', NULL, 'pwikit-demo', DEFAULT);

INSERT INTO public.web_article_authors VALUES (1, 1, 1);
INSERT INTO public.web_article_authors VALUES (2, 2, 1);
INSERT INTO public.web_article_authors VALUES (3, 3, 1);
INSERT INTO public.web_article_authors VALUES (4, 4, 1);
INSERT INTO public.web_article_authors VALUES (5, 5, 1);
INSERT INTO public.web_article_authors VALUES (6, 11, 30);
INSERT INTO public.web_article_authors VALUES (7, 12, 30);
INSERT INTO public.web_article_authors VALUES (8, 14, 30);
INSERT INTO public.web_article_authors VALUES (9, 12, 31);
INSERT INTO public.web_article_authors VALUES (10, 15, 30);
INSERT INTO public.web_article_authors VALUES (17, 135, 30);
INSERT INTO public.web_article_authors VALUES (19, 137, 30);
INSERT INTO public.web_article_authors VALUES (20, 138, 30);
INSERT INTO public.web_article_authors VALUES (21, 139, 30);
INSERT INTO public.web_article_authors VALUES (30, 198, 30);

INSERT INTO public.web_tagscategory VALUES (1, '默认', '', NULL, '_default');
INSERT INTO public.web_tagscategory VALUES (2, 'lang', '', 1, 'lang');
INSERT INTO public.web_tagscategory VALUES (3, 'Probe Topic', 'what a page is about', 2, 'topic');

INSERT INTO public.web_tag VALUES (1, 'zeta', 1);
INSERT INTO public.web_tag VALUES (2, 'alpha', 1);
INSERT INTO public.web_tag VALUES (3, 'en', 2);
INSERT INTO public.web_tag VALUES (4, 'scp', 3);
INSERT INTO public.web_tag VALUES (5, '_staff', 1);
INSERT INTO public.web_tag VALUES (6, 'aaa', 2);

INSERT INTO public.web_article_tags VALUES (1, 12, 1);
INSERT INTO public.web_article_tags VALUES (2, 12, 2);
INSERT INTO public.web_article_tags VALUES (3, 12, 3);
INSERT INTO public.web_article_tags VALUES (4, 14, 2);
INSERT INTO public.web_article_tags VALUES (5, 14, 4);
INSERT INTO public.web_article_tags VALUES (6, 135, 2);
INSERT INTO public.web_article_tags VALUES (7, 135, 5);
INSERT INTO public.web_article_tags VALUES (8, 137, 2);
INSERT INTO public.web_article_tags VALUES (9, 137, 4);
INSERT INTO public.web_article_tags VALUES (10, 139, 1);
INSERT INTO public.web_article_tags VALUES (11, 139, 5);
INSERT INTO public.web_article_tags VALUES (12, 138, 4);
INSERT INTO public.web_article_tags VALUES (13, 138, 6);

INSERT INTO public.web_articlelogentry VALUES (5, 'new', '{"title": "scp-173", "version_id": 5}', '2024-11-12 13:18:15+00', 'seed', 0, 5, 1);
INSERT INTO public.web_articlelogentry VALUES (6, 'title', '{"title": "Shared Component", "prev_title": "probe-var"}', '2024-11-12 13:19:15+00', '', 0, 8, NULL);
INSERT INTO public.web_articlelogentry VALUES (12, 'new', '{"title": "parent", "version_id": 11}', '2024-11-12 13:25:15+00', '', 0, 11, 30);
INSERT INTO public.web_articlelogentry VALUES (16, 'new', '{"title": "half", "version_id": 15}', '2024-11-12 13:29:15+00', '', 0, 15, 30);
INSERT INTO public.web_articlelogentry VALUES (17, 'new', '{"title": "included", "version_id": 16}', '2024-11-12 13:30:15+00', '', 0, 16, NULL);
INSERT INTO public.web_articlelogentry VALUES (21, 'new', '{"title": "described", "version_id": 20}', '2024-11-12 13:33:15+00', '', 0, 113, NULL);
INSERT INTO public.web_articlelogentry VALUES (22, 'new', '{"title": "imaged", "version_id": 21}', '2024-11-12 13:34:15+00', '', 0, 114, NULL);
INSERT INTO public.web_articlelogentry VALUES (23, 'new', '{"title": "tagged", "version_id": 22}', '2024-11-12 13:35:15+00', '', 0, 115, NULL);
INSERT INTO public.web_articlelogentry VALUES (25, 'new', '{"title": "unknownmodule", "version_id": 24}', '2024-11-12 13:37:15+00', '', 0, 117, NULL);
INSERT INTO public.web_articlelogentry VALUES (26, 'source', '{"version_id": 25}', '2024-11-12 13:38:15+00', 'Seeding', 1, 4, NULL);
INSERT INTO public.web_articlelogentry VALUES (27, 'new', '{"title": "wiki-syntax-guide", "version_id": 26}', '2024-11-12 13:39:15+00', 'Seeding', 0, 118, NULL);
INSERT INTO public.web_articlelogentry VALUES (29, 'new', '{"title": "recent-posts", "version_id": 28}', '2024-11-12 13:41:15+00', 'Seeding', 0, 120, NULL);
INSERT INTO public.web_articlelogentry VALUES (42, 'new', '{"title": "listtags", "version_id": 41}', '2024-11-12 13:54:15+00', '', 0, 131, NULL);
INSERT INTO public.web_articlelogentry VALUES (43, 'new', '{"title": "listempty", "version_id": 42}', '2024-11-12 13:55:15+00', '', 0, 132, NULL);
INSERT INTO public.web_articlelogentry VALUES (44, 'new', '{"title": "listurl", "version_id": 43}', '2024-11-12 13:56:15+00', '', 0, 133, NULL);
INSERT INTO public.web_articlelogentry VALUES (45, 'new', '{"title": "listbyvotes", "version_id": 44}', '2024-11-12 13:57:15+00', '', 0, 134, NULL);
INSERT INTO public.web_articlelogentry VALUES (46, 'new', '{"title": "unrated", "version_id": 45}', '2024-11-12 13:58:15+00', '', 0, 135, 30);
INSERT INTO public.web_articlelogentry VALUES (48, 'new', '{"title": "quarter", "version_id": 47}', '2024-11-12 13:59:15+00', '', 0, 137, 30);
INSERT INTO public.web_articlelogentry VALUES (49, 'new', '{"title": "unratable", "version_id": 48}', '2024-11-12 14:00:15+00', '', 0, 138, 30);
INSERT INTO public.web_articlelogentry VALUES (50, 'new', '{"title": "third", "version_id": 49}', '2024-11-12 14:01:15+00', '', 0, 139, 30);
INSERT INTO public.web_articlelogentry VALUES (51, 'new', '{"title": "styled", "version_id": 50}', '2024-11-12 14:02:15+00', '', 0, 196, NULL);
INSERT INTO public.web_articlelogentry VALUES (52, 'new', '{"title": "styledhead", "version_id": 51}', '2024-11-12 14:03:15+00', '', 0, 197, NULL);
INSERT INTO public.web_articlelogentry VALUES (53, 'new', '{"title": "changes", "version_id": 52}', '2024-11-12 14:04:15+00', '', 0, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (241, 'source', '{"version_id": 0}', '2024-11-12 14:05:15+00', 'a source edit', 1, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (242, 'source', '{"version_id": 0}', '2024-11-12 14:06:15+00', '   ', 2, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (243, 'source', '{"version_id": 0}', '2024-11-12 14:07:15+00', '', 3, 198, NULL);
INSERT INTO public.web_articlelogentry VALUES (244, 'source', '{"version_id": 0}', '2024-11-12 14:08:15+00', '', 4, 198, 31);
INSERT INTO public.web_articlelogentry VALUES (245, 'source', '{"version_id": 0}', '2024-11-12 14:09:15+00', '', 5, 198, 32);
INSERT INTO public.web_articlelogentry VALUES (246, 'title', '{"title": "Probe Changes", "prev_title": "Probe \"Old\" <b>"}', '2024-11-12 14:10:15+00', '', 6, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (247, 'name', '{"name": "probe:changes", "prev_name": "probe:was-here"}', '2024-11-12 14:11:15+00', '', 7, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (248, 'tags', '{"added_tags": [{"id": 1, "name": "alpha"}], "removed_tags": []}', '2024-11-12 14:12:15+00', '', 8, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (249, 'tags', '{"added_tags": [], "removed_tags": [{"id": 2, "name": "lang:en"}]}', '2024-11-12 14:13:15+00', '', 9, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (2, 'new', '{"title": "side", "version_id": 2}', '2024-11-12 13:15:15+00', 'seed', 0, 2, 1);
INSERT INTO public.web_articlelogentry VALUES (3, 'new', '{"title": "box", "version_id": 3}', '2024-11-12 13:16:15+00', 'seed', 0, 3, 1);
INSERT INTO public.web_articlelogentry VALUES (4, 'new', '{"title": "main", "version_id": 4}', '2024-11-12 13:17:15+00', 'seed', 0, 4, 1);
INSERT INTO public.web_articlelogentry VALUES (7, 'new', '{"title": "Shared Component", "version_id": 8}', '2024-11-12 13:20:15+00', '', 1, 8, NULL);
INSERT INTO public.web_articlelogentry VALUES (8, 'title', '{"title": "Page A", "prev_title": "probe-a"}', '2024-11-12 13:21:15+00', '', 0, 9, NULL);
INSERT INTO public.web_articlelogentry VALUES (9, 'new', '{"title": "Page A", "version_id": 9}', '2024-11-12 13:22:15+00', '', 1, 9, NULL);
INSERT INTO public.web_articlelogentry VALUES (10, 'title', '{"title": "Page B", "prev_title": "probe-b"}', '2024-11-12 13:23:15+00', '', 0, 10, NULL);
INSERT INTO public.web_articlelogentry VALUES (11, 'new', '{"title": "Page B", "version_id": 10}', '2024-11-12 13:24:15+00', '', 1, 10, NULL);
INSERT INTO public.web_articlelogentry VALUES (13, 'new', '{"title": "full", "version_id": 12}', '2024-11-12 13:26:15+00', '', 0, 12, 30);
INSERT INTO public.web_articlelogentry VALUES (14, 'new', '{"title": "bare", "version_id": 13}', '2024-11-12 13:27:15+00', '', 0, 13, NULL);
INSERT INTO public.web_articlelogentry VALUES (15, 'new', '{"title": "rated", "version_id": 14}', '2024-11-12 13:28:15+00', '', 0, 14, 30);
INSERT INTO public.web_articlelogentry VALUES (18, 'new', '{"title": "host", "version_id": 17}', '2024-11-12 13:31:15+00', '', 0, 17, NULL);
INSERT INTO public.web_articlelogentry VALUES (20, 'new', '{"title": "redirect", "version_id": 19}', '2024-11-12 13:32:15+00', '', 0, 112, NULL);
INSERT INTO public.web_articlelogentry VALUES (24, 'new', '{"title": "taggedplain", "version_id": 23}', '2024-11-12 13:36:15+00', '', 0, 116, NULL);
INSERT INTO public.web_articlelogentry VALUES (28, 'new', '{"title": "site", "version_id": 27}', '2024-11-12 13:40:15+00', 'Seeding', 0, 119, NULL);
INSERT INTO public.web_articlelogentry VALUES (1, 'new', '{"title": "top", "version_id": 1}', '2024-11-12 13:14:15+00', 'seed', 0, 1, 1);
INSERT INTO public.web_articlelogentry VALUES (30, 'new', '{"title": "new-thread", "version_id": 29}', '2024-11-12 13:42:15+00', 'Seeding', 0, 121, NULL);
INSERT INTO public.web_articlelogentry VALUES (31, 'new', '{"title": "category", "version_id": 30}', '2024-11-12 13:43:15+00', 'Seeding', 0, 122, NULL);
INSERT INTO public.web_articlelogentry VALUES (32, 'new', '{"title": "thread", "version_id": 31}', '2024-11-12 13:44:15+00', 'Seeding', 0, 123, NULL);
INSERT INTO public.web_articlelogentry VALUES (33, 'new', '{"title": "start", "version_id": 32}', '2024-11-12 13:45:15+00', 'Seeding', 0, 124, NULL);
INSERT INTO public.web_articlelogentry VALUES (262, 'votes_deleted', '{"rating": 0, "popularity": 0, "rating_mode": "disabled", "votes_count": 0}', '2024-11-12 14:26:15+00', '', 22, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (34, 'new', '{"title": "top-impl", "version_id": 33}', '2024-11-12 13:46:15+00', 'Seeding', 0, 125, NULL);
INSERT INTO public.web_articlelogentry VALUES (35, 'source', '{"version_id": 34}', '2024-11-12 13:47:15+00', 'Seeding', 1, 1, NULL);
INSERT INTO public.web_articlelogentry VALUES (36, 'source', '{"version_id": 35}', '2024-11-12 13:48:15+00', 'Seeding', 1, 2, NULL);
INSERT INTO public.web_articlelogentry VALUES (37, 'new', '{"title": "bydisplay", "version_id": 36}', '2024-11-12 13:49:15+00', '', 0, 126, NULL);
INSERT INTO public.web_articlelogentry VALUES (38, 'new', '{"title": "listed", "version_id": 37}', '2024-11-12 13:50:15+00', '', 0, 127, NULL);
INSERT INTO public.web_articlelogentry VALUES (39, 'new', '{"title": "listjoined", "version_id": 38}', '2024-11-12 13:51:15+00', '', 0, 128, NULL);
INSERT INTO public.web_articlelogentry VALUES (40, 'new', '{"title": "listnowrap", "version_id": 39}', '2024-11-12 13:52:15+00', '', 0, 129, NULL);
INSERT INTO public.web_articlelogentry VALUES (41, 'new', '{"title": "listsections", "version_id": 40}', '2024-11-12 13:53:15+00', '', 0, 130, NULL);
INSERT INTO public.web_articlelogentry VALUES (250, 'tags', '{"added_tags": [{"id": 1, "name": "alpha"}, {"id": 3, "name": "Zeta"}], "removed_tags": [{"id": 2, "name": "lang:en"}]}', '2024-11-12 14:14:15+00', '', 10, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (251, 'tags', '{}', '2024-11-12 14:15:15+00', '', 11, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (252, 'parent', '{"parent": "probe:parent", "prev_parent": null}', '2024-11-12 14:16:15+00', '', 12, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (253, 'parent', '{"parent": null, "prev_parent": "probe:parent"}', '2024-11-12 14:17:15+00', '', 13, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (254, 'parent', '{"parent": "probe:full", "prev_parent": "probe:parent"}', '2024-11-12 14:18:15+00', '', 14, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (255, 'parent', '{"parent": null, "prev_parent": null}', '2024-11-12 14:19:15+00', '', 15, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (256, 'file_added', '{"id": 1, "name": "cover.png"}', '2024-11-12 14:20:15+00', '', 16, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (257, 'file_deleted', '{"id": 1, "name": "cover.png"}', '2024-11-12 14:21:15+00', '', 17, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (258, 'file_renamed', '{"name": "banner.png", "prev_name": "cover.png"}', '2024-11-12 14:22:15+00', '', 18, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (259, 'votes_deleted', '{"rating": 3, "popularity": 60, "rating_mode": "updown", "votes_count": 5}', '2024-11-12 14:23:15+00', '', 19, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (260, 'votes_deleted', '{"rating": -3.7, "popularity": 11, "rating_mode": "updown", "votes_count": 9}', '2024-11-12 14:24:15+00', '', 20, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (261, 'votes_deleted', '{"rating": 4.25, "popularity": 75, "rating_mode": "stars", "votes_count": 4}', '2024-11-12 14:25:15+00', '', 21, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (263, 'authorship', '{"added_authors": [30], "removed_authors": []}', '2024-11-12 14:27:15+00', '', 23, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (264, 'authorship', '{"added_authors": [30, 32], "removed_authors": []}', '2024-11-12 14:28:15+00', '', 24, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (265, 'authorship', '{"added_authors": [], "removed_authors": [31]}', '2024-11-12 14:29:15+00', '', 25, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (266, 'authorship', '{"added_authors": [32], "removed_authors": [31]}', '2024-11-12 14:30:15+00', '', 26, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (267, 'authorship', '{}', '2024-11-12 14:31:15+00', '', 27, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (268, 'wikidot', '{}', '2024-11-12 14:32:15+00', 'imported from wikidot', 28, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (269, 'revert', '{"subtypes": ["source", "title"], "rev_number": 2}', '2024-11-12 14:33:15+00', '', 29, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (270, 'revert', '{"subtypes": [], "rev_number": 0}', '2024-11-12 14:34:15+00', '', 30, 198, 30);
INSERT INTO public.web_articlelogentry VALUES (271, 'revert', '{"rev_number": 1}', '2024-11-12 14:35:15+00', '', 31, 198, 30);

INSERT INTO public.web_articlesearchindex VALUES (1, '[[module ForumCategory]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '[[module ForumCategory]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '''forumcategori'':2,6 ''modul'':1,5 ''如果您希望论坛正常工作'':3,7 ''请不要更改此页面'':4,8', 122);
INSERT INTO public.web_articlesearchindex VALUES (2, 'quarter source
[[[wanted:beta]]] [[[probe:no-such-one]]]', 'quarter source
[[[wanted:beta]]] [[[probe:no-such-one]]]', '''beta'':4,13 ''no-such-on'':6,15 ''one'':9,18 ''probe'':5,14 ''quarter'':1,10 ''sourc'':2,11 ''want'':3,12', 137);
INSERT INTO public.web_articlesearchindex VALUES (3, '[[*user Probe WD]]', '[[*user Probe WD]]', '''probe'':2,5 ''user'':1,4 ''wd'':3,6', 126);
INSERT INTO public.web_articlesearchindex VALUES (4, '[[module ListPages category="probe" name="no-such-name-at-all"]]
%%name%%
[[/module]]', '[[module ListPages category="probe" name="no-such-name-at-all"]]
%%name%%
[[/module]]', '''/module'':13,26 ''categori'':3,16 ''listpag'':2,15 ''modul'':1,14 ''name'':5,9,12,18,22,25 ''no-such-name-at-al'':6,19 ''probe'':4,17', 132);
INSERT INTO public.web_articlesearchindex VALUES (5, '[[div class="box"]]
这是一个被 include 的组件。参数 a = %%a%%
[[/div]]', '[[div class="box"]]
这是一个被 include 的组件。参数 a = %%a%%
[[/div]]', '''/div'':10,20 ''box'':3,13 ''class'':2,12 ''div'':1,11 ''includ'':5,15 ''参数'':7,17 ''的组件'':6,16 ''这是一个被'':4,14', 3);
INSERT INTO public.web_articlesearchindex VALUES (6, '[[module PagesByTag tag="lang:en"]]', '[[module PagesByTag tag="lang:en"]]', '''en'':5,10 ''lang'':4,9 ''modul'':1,6 ''pagesbytag'':2,7 ''tag'':3,8', 115);
INSERT INTO public.web_articlesearchindex VALUES (7, '[[module ListPages category="probe" order="name" wrapper="no" limit="2"]]
%%name%%
[[/module]]', '[[module ListPages category="probe" order="name" wrapper="no" limit="2"]]
%%name%%
[[/module]]', '''/module'':12,24 ''2'':10,22 ''categori'':3,15 ''limit'':9,21 ''listpag'':2,14 ''modul'':1,13 ''name'':6,11,18,23 ''order'':5,17 ''probe'':4,16 ''wrapper'':7,19', 129);
INSERT INTO public.web_articlesearchindex VALUES (8, '[[module ForumThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '[[module ForumThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '''forumthread'':2,6 ''modul'':1,5 ''如果您希望论坛正常工作'':3,7 ''请不要更改此页面'':4,8', 123);
INSERT INTO public.web_articlesearchindex VALUES (9, '[[include component:box a=173]]

**项目编号：** SCP-173

**项目等级：** Euclid

+ 描述

一个测试用条目，包含 [[[main | 内链]]]、[[[another-missing | 红链]]] 和一个代码块：

[[code type="python"]]
print("hello")
[[/code]]

[[module Rate]]', '[[include component:box a=173]]

**项目编号：** SCP-173

**项目等级：** Euclid

+ 描述

一个测试用条目，包含 [[[main | 内链]]]、[[[another-missing | 红链]]] 和一个代码块：

[[code type="python"]]
print("hello")
[[/code]]

[[module Rate]]', '''-173'':8,36 ''/code'':26,54 ''173'':5,33 ''anoth'':17,45 ''another-miss'':16,44 ''box'':3,31 ''code'':21,49 ''compon'':2,30 ''euclid'':10,38 ''hello'':25,53 ''includ'':1,29 ''main'':14,42 ''miss'':18,46 ''modul'':27,55 ''print'':24,52 ''python'':23,51 ''rate'':28,56 ''scp'':7,35 ''type'':22,50 ''一个测试用条目'':12,40 ''内链'':15,43 ''包含'':13,41 ''和一个代码块'':20,48 ''描述'':11,39 ''红链'':19,47 ''项目等级'':9,37 ''项目编号'':6,34', 5);
INSERT INTO public.web_articlesearchindex VALUES (10, 'bare source', 'bare source', '''bare'':1,3 ''sourc'':2,4', 13);
INSERT INTO public.web_articlesearchindex VALUES (11, '本组件被谁包含: %%this|title%% / %%this|fullname%% / 评分 %%this|rating%%', '本组件被谁包含: %%this|title%% / %%this|fullname%% / 评分 %%this|rating%%', '''fullnam'':5,13 ''rate'':8,16 ''titl'':3,11 ''本组件被谁包含'':1,9 ''评分'':6,14', 8);
INSERT INTO public.web_articlesearchindex VALUES (12, '[[include probe:included]]', '[[include probe:included]]', '''includ'':1,3,4,6 ''probe'':2,5', 17);
INSERT INTO public.web_articlesearchindex VALUES (13, 'before
[[module Redirect destination="/probe:full"]]
after', 'before
[[module Redirect destination="/probe:full"]]
after', '''/probe'':5,11 ''destin'':4,10 ''full'':6,12 ''modul'':2,8 ''redirect'':3,9', 112);
INSERT INTO public.web_articlesearchindex VALUES (14, '[[include component:probe-var]]', '[[include component:probe-var]]', '''compon'':2,7 ''includ'':1,6 ''probe'':4,9 ''probe-var'':3,8 ''var'':5,10', 9);
INSERT INTO public.web_articlesearchindex VALUES (15, '[[include component:probe-var]]', '[[include component:probe-var]]', '''compon'':2,7 ''includ'':1,6 ''probe'':4,9 ''probe-var'':3,8 ''var'':5,10', 10);
INSERT INTO public.web_articlesearchindex VALUES (16, '[[module PagesByTag tag="zeta"]]', '[[module PagesByTag tag="zeta"]]', '''modul'':1,5 ''pagesbytag'':2,6 ''tag'':3,7 ''zeta'':4,8', 116);
INSERT INTO public.web_articlesearchindex VALUES (17, '[[div class="new-post"]]
[[[forum:recent-posts|论坛新帖]]]
[[/div]]

[[module ForumStart]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]', '[[div class="new-post"]]
[[[forum:recent-posts|论坛新帖]]]
[[/div]]

[[module ForumStart]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]', '''/div'':11,26 ''class'':2,17 ''div'':1,16 ''forum'':6,21 ''forumstart'':13,28 ''modul'':12,27 ''new'':4,19 ''new-post'':3,18 ''post'':5,9,20,24 ''recent'':8,23 ''recent-post'':7,22 ''如果您希望论坛正常工作'':14,29 ''论坛新帖'':10,25 ''请不要更改此页面'':15,30', 124);
INSERT INTO public.web_articlesearchindex VALUES (18, 'unratable source
[[[wanted:gamma]]] [[[probe:full|a page that exists]]]', 'unratable source
[[[wanted:gamma]]] [[[probe:full|a page that exists]]]', '''exist'':10,20 ''full'':6,16 ''gamma'':4,14 ''page'':8,18 ''probe'':5,15 ''sourc'':2,12 ''unrat'':1,11 ''want'':3,13', 138);
INSERT INTO public.web_articlesearchindex VALUES (19, '* [# 这里]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 是]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 一个]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 示例]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 顶部栏]
 * [https://github.com/WikitTeam/ProjectWikit GitHub页面]
 * [[[/forum/start|论坛]]]
 * [[[/forum:recent-posts|最新帖子]]]
 * [[[/wiki-syntax-guide|维基语法指南]]]
', '* [# 这里]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 是]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 一个]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 示例]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 顶部栏]
 * [https://github.com/WikitTeam/ProjectWikit GitHub页面]
 * [[[/forum/start|论坛]]]
 * [[[/forum:recent-posts|最新帖子]]]
 * [[[/wiki-syntax-guide|维基语法指南]]]
', '''/forum'':48,102 ''/forum/start'':46,100 ''/wiki-syntax-guide'':53,107 ''/wikitteam/projectwikit'':44,98 ''github.com'':43,97 ''github.com/wikitteam/projectwikit'':42,96 ''github页面'':45,99 ''main'':2,4,6,8,10,13,15,17,19,22,24,26,28,30,32,35,37,39,56,58,60,62,64,67,69,71,73,76,78,80,82,84,86,89,91,93 ''post'':51,105 ''recent'':50,104 ''recent-post'':49,103 ''一个'':21,75 ''是'':12,66 ''最新帖子'':52,106 ''示例'':34,88 ''维基语法指南'':54,108 ''论坛'':47,101 ''这里'':1,55 ''顶部栏'':41,95 ''鱼'':3,5,7,9,11,14,16,18,20,23,25,27,29,31,33,36,38,40,57,59,61,63,65,68,70,72,74,77,79,81,83,85,87,90,92,94', 125);
INSERT INTO public.web_articlesearchindex VALUES (20, '[[div class="top-bar"]]
[[include nav:top-impl]]
[[/div]]

[[div class="mobile-top-bar"]]
[[div class="open-menu"]]
[#side-bar ≡]
[[/div]]
[[include nav:top-impl]]
[[/div]]', '[[div class="top-bar"]]
[[include nav:top-impl]]
[[/div]]

[[div class="mobile-top-bar"]]
[[div class="open-menu"]]
[#side-bar ≡]
[[/div]]
[[include nav:top-impl]]
[[/div]]', '''/div'':11,26,32,43,58,64 ''bar'':5,17,25,37,49,57 ''class'':2,13,19,34,45,51 ''div'':1,12,18,33,44,50 ''impl'':10,31,42,63 ''includ'':6,27,38,59 ''menu'':22,54 ''mobil'':15,47 ''mobile-top-bar'':14,46 ''nav'':7,28,39,60 ''open'':21,53 ''open-menu'':20,52 ''side'':24,56 ''side-bar'':23,55 ''top'':4,9,16,30,36,41,48,62 ''top-bar'':3,35 ''top-impl'':8,29,40,61', 1);
INSERT INTO public.web_articlesearchindex VALUES (21, '++ 存在的页面：

[[module listpages category="*" separate="False" prependLine="|| **标题** || **名称** ||"]]
|| %%title_linked%% || %%fullname%% ||
[[/module]]', '++ 存在的页面：

[[module listpages category="*" separate="False" prependLine="|| **标题** || **名称** ||"]]
|| %%title_linked%% || %%fullname%% ||
[[/module]]', '''/module'':13,26 ''categori'':4,17 ''fals'':6,19 ''fullnam'':12,25 ''link'':11,24 ''listpag'':3,16 ''modul'':2,15 ''prependlin'':7,20 ''separ'':5,18 ''titl'':10,23 ''名称'':9,22 ''存在的页面'':1,14 ''标题'':8,21', 2);
INSERT INTO public.web_articlesearchindex VALUES (22, '[[module SiteChanges]]', '[[module SiteChanges]]', '''modul'':1,3 ''sitechang'':2,4', 198);
INSERT INTO public.web_articlesearchindex VALUES (23, '[[module ListPages category="probe" order="name" perPage="2"]]
[[head]]
top
[[/head]]
[[body]]
%%name%%
[[/body]]
[[foot]]
bottom
[[/foot]]
[[/module]]', '[[module ListPages category="probe" order="name" perPage="2"]]
[[head]]
top
[[/head]]
[[body]]
%%name%%
[[/body]]
[[foot]]
bottom
[[/foot]]
[[/module]]', '''/body'':14,32 ''/foot'':17,35 ''/head'':11,29 ''/module'':18,36 ''2'':8,26 ''bodi'':12,30 ''bottom'':16,34 ''categori'':3,21 ''foot'':15,33 ''head'':9,27 ''listpag'':2,20 ''modul'':1,19 ''name'':6,13,24,31 ''order'':5,23 ''perpag'':7,25 ''probe'':4,22 ''top'':10,28', 130);
INSERT INTO public.web_articlesearchindex VALUES (24, '[[module ListPages category="*" tags="+lang:en -zeta" order="fullname"]]
%%fullname%%
[[/module]]', '[[module ListPages category="*" tags="+lang:en -zeta" order="fullname"]]
%%fullname%%
[[/module]]', '''/module'':11,22 ''categori'':3,14 ''en'':6,17 ''fullnam'':9,10,20,21 ''lang'':5,16 ''listpag'':2,13 ''modul'':1,12 ''order'':8,19 ''tag'':4,15 ''zeta'':7,18', 131);
INSERT INTO public.web_articlesearchindex VALUES (25, '[[module ListPages category="probe" order="name" perPage="3" separate="yes"]]
%%index%%/%%total%% [[[%%fullname%%|%%title%%]]] %%rating%%
[[/module]]', '[[module ListPages category="probe" order="name" perPage="3" separate="yes"]]
%%index%%/%%total%% [[[%%fullname%%|%%title%%]]] %%rating%%
[[/module]]', '''/module'':16,32 ''3'':8,24 ''categori'':3,19 ''fullnam'':13,29 ''index'':11,27 ''listpag'':2,18 ''modul'':1,17 ''name'':6,22 ''order'':5,21 ''perpag'':7,23 ''probe'':4,20 ''rate'':15,31 ''separ'':9,25 ''titl'':14,30 ''total'':12,28 ''yes'':10,26', 127);
INSERT INTO public.web_articlesearchindex VALUES (26, '[[module CSS]]
#page-content { color : red ; }
@media (max-width: 767px) { #main { padding : 0 ; } }
[[/module]]
styled body', '[[module CSS]]
#page-content { color : red ; }
@media (max-width: 767px) { #main { padding : 0 ; } }
[[/module]]
styled body', '''/module'':16,34 ''0'':15,33 ''767px'':12,30 ''bodi'':18,36 ''color'':6,24 ''content'':5,23 ''css'':2,20 ''main'':13,31 ''max'':10,28 ''max-width'':9,27 ''media'':8,26 ''modul'':1,19 ''pad'':14,32 ''page'':4,22 ''page-cont'':3,21 ''red'':7,25 ''style'':17,35 ''width'':11,29', 196);
INSERT INTO public.web_articlesearchindex VALUES (27, 'visible text
[[module PageDescription]]custom description[[/module]]', 'visible text
[[module PageDescription]]custom description[[/module]]', '''/module'':7,14 ''custom'':5,12 ''descript'':6,13 ''modul'':3,10 ''pagedescript'':4,11 ''text'':2,9 ''visibl'':1,8', 113);
INSERT INTO public.web_articlesearchindex VALUES (28, 'rated source', 'rated source', '''rate'':1,3 ''sourc'':2,4', 14);
INSERT INTO public.web_articlesearchindex VALUES (29, '+ 恭喜！一切正常！', '+ 恭喜！一切正常！', '''一切正常'':2,4 ''恭喜'':1,3', 4);
INSERT INTO public.web_articlesearchindex VALUES (30, '[[module PageImage src="probe:full/cover.png"]]body text', '[[module PageImage src="probe:full/cover.png"]]body text', '''bodi'':6,13 ''full/cover.png'':5,12 ''modul'':1,8 ''pageimag'':2,9 ''probe'':4,11 ''src'':3,10 ''text'':7,14', 114);
INSERT INTO public.web_articlesearchindex VALUES (31, '以下指南是关于PojectWikit网站整体运行机制以及维基标记语言（wiki markup）相关内容的技术文档。

本指南面向具备最低限度 HTML 和 CSS 基础知识的读者；其目标是全面、详尽地描述所有可用功能，并解释为什么某些内容会以目前这种方式运行。

[[toc]]

[[module CSS]]

#page-content h1, #page-content h2, #page-content h3 {
  padding-bottom: 8px;
  border-bottom: 1px solid #aaa;
  margin-top: 32px;
  clear: both;
}

#page-content h1 + h2, #page-content h2 + h3 {
  margin-top: 16px;
}

code, .code, .code pre {
  background: #f7f7f7;
  color: #050;
  font-weight: 500;
  font-family: ''Cascadia Mono'', ''Courier New'', Courier, FreeMono, monospace;
}

code {
  padding: 4px;
  border-radius: 4px;
  white-space: nowrap;
}

.code {
  padding: 8px;
}

.code p, .code pre {
  margin: 0;
}

#page-content dl {
  display: grid;
  grid-template-columns: max-content max-content;
  border: 1px solid #eee;
  border-radius: 8px;
  float: right;
  overflow: hidden;
  margin-left: 32px;
  margin-bottom: 32px;
  background: white;
}

#page-content dl dd, #page-content dl dt {
  padding: 8px;
  border-bottom: 1px solid #eee;
  margin: 0;
}

#page-content dl dd {
  text-align: right;
}

.actual-page-content a[href^="#"] {
  border-bottom: 1px dotted #050;
  color: #050;
  text-decoration: none;
}

.actual-page-content a[href^="#"]:hover {
  border-bottom-style: solid;
}
  @media (max-width: 700px) {
code {
  white-space: wrap;
  word-break: break-all;
}
  }
[[/module]]

[[div class="actual-page-content"]]

+ 引言

标记语言分为四种类型：

* **自动替换** 在文章开始处理之前执行。因此，自动替换允许在文章中添加新的代码，这些代码随后会作为语法被处理。详见 {{[[include]]}} 章节。

* **段落划分** 按既定规则自动进行。

* **自由语法** 没有严格的格式；每个标记元素可能以完全不可预测的方式被解析。这些元素不总是彼此兼容，也不一定与块级元素兼容。

* **块级元素** 具有严格规则；其格式在外观上类似于 HTML 标记或 BBCode。块级元素具有名称、属性、修饰符。那些原则上可以包含其他元素的块级元素，对其所包含元素的类型不作限制（包括其他块级元素）。

+ [[# autoreplace]] 自动替换

++ {{[[include]]}}：从其他文章插入代码

语法：

[[div class="code"]]
@@[[include 文章名称 参数1 = 值1 | 参数2 = 值2]]@@
[[/div]]

为了使该元素正常工作，在起始的 {{[[}} 前面从行首开始不得有任何文本（包括空格）。同样，在结束的 {{]]}} 之后也不能有任何文本。

元素 {{[[include]]}}，以及其中的各个参数，都可以占用多行。

使用该元素时，系统会访问指定的站点文章，获取其源代码，并将该源代码插入到 {{[[include]]}} 所在的位置。

在插入之前，会对指定文章中的所有变量进行自动替换。例如，形如 {{@@{$参数1}@@}} 的变量将被替换为在该元素中指定的对应参数值。

由于插入源代码是在“行级”而非“元素级”进行的，因此被嵌入的文章中可以包含完整或部分源代码。同样，{{[[include]]}} 的参数中也可以包含完整或部分源代码。

示例：

* 文章 {{page1}} 中的代码： _
[[div class="code"]]
@@{$param}@@
[[/div]]

* 使用 {{[[include]]}} 的文章中的代码： _
[[div class="code"]]
@@[[include page1 param=[[div class="code"]] ]]@@
@@text@@
@@[[include page1 param=[[/div]] ]]@@
[[/div]]

* 结果： _
[[div class="code"]]
@@[[div class="code"]]@@
@@text@@
@@[[/div]]@@
[[/div]]

++ {{[[noinclude]]}}：在被插入到其他页面时忽略部分代码

语法：

[[div class="code"]]
@@[[noinclude]]@@
...任意文本...
@@[[/noinclude]]@@
[[/div]]

某些站点组件同时包含可调用代码（组件本身）、使用说明文档以及预览内容。

为了防止这些可视化元素被包含到作者文章中，可以使用 {{[[noinclude]]}} 标签。

无论是起始还是结束的 {{[[noinclude]]}} 标签，都必须单独占据一整行，否则标签不会生效。这样设计是为了降低误触发或错误触发的概率，例如在记录该功能自身文档时。

++ 分类模板

分类模板是形如 [[[component:_template|component:_template]]] 的隐藏页面。对于主分类，页面名称为 [[[_default:_template|_template]]].

如果为某个分类（例如此处的 {{component}}）指定了模板，那么该模板将会为该分类下的所有文章渲染显示，**而不是文章的实际代码**。同时，模板中支持 [#module-listpages ListPages 模块] 中使用的所有变量。例如，可以通过 {{%%content%%}} 获取原始文章代码。

++ [[# path-params]] {{%%path%%}}, {{%%path_expr%%}}, {{%%path_url%%}}：页面参数

站点引擎支持通过形如 {{/参数/值}} 的语法在页面地址中传递参数。

例如，为了在文章 {{page1}} 中访问参数 {{%%param%%}} 和 {{%%param2%%}}，可以通过如下地址访问：

{{@@https://projwikit.unitreaty.org/page1/param/example1/param2/example2@@}}

由于这些变量替换属于自动替换，目标文章可以通过三种方式访问参数：

* {{%%path|param%%}} 直接将变量值插入文章代码；如果未指定该变量，则插入文本 {{%%path|param%%}}。

* {{%%path_expr|param%%}} 以 JSON 字符串格式插入变量值；若未指定，则插入文本 {{"%%path_expr|param%%"}}。这允许在块级元素属性中传递包含特殊字符的复杂值（例如 {{@@[[input type="text" value=%%path_expr|param%%]]@@}} 可确保即便用户使用特殊字符，值也能正确写入）。

* {{%%path_url|param%%}} 插入 URL 编码格式的变量值；若未指定，则插入 {{%25%25path_url%7Cparam%25%25}}。这允许在链接或传递给其他页面的参数中使用这些值（例如 {{@@[[module Redirect to="/other_page/param/%%path_url|param%%"]]@@}}）。

++ 排版符号的自动替换

* {{@<&#96;>@文本@<&#39;>@}} —— 替换为 ‘文本’。

* {{@<&#96;>@@<&#96;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 “文本”。

* {{@<&#44;>@@<&#44;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 „文本”。

[!-- * {{@<&#46;>@@<&#46;>@@<&#46;>@}}, {{@<&#46;>@ @<&#46;>@ @<&#46;>@}} —— 替换为符号 "…". --] [!-- 暂时移除 // jewalky --]

需要注意的是，由于这些符号替换发生在自动替换阶段，因此可能跨多行发生，甚至包括在 [#literals 字面量]、{{@@[[code]]@@}}、{{@@[[module]]@@}} 等内容中；请务必注意。

+ 段落划分

系统中所有可以包含其他元素的元素，分为两大类：

* 行内元素。包括普通文本、所有文本格式元素，以及 {{@@[[span]]@@}} 和其他诸如 {{@@[[image]]@@}}、{{@@[[user]]@@}} 等元素。一般来说，如果该元素默认显示为 {{display: inline}} 或 {{display: inline-block}}，则可视为行内元素。

* 全宽元素。包括标题、分隔线、列表、{{@@[[toc]]@@}}、{{@@[[div]]@@}}、{{@@[[blockquote]]@@}}、{{@@[[footnoteblock]]@@}}、{{@@[[collapsible]]@@}} 等。大致对应 {{display: block}}。

尽管上文提及 CSS 属性，但该属性的实际值不会影响段落生成，因为元素的分类是在其从标记转换为 HTML 的初始阶段完成的。

段落会被创建：

* 在全宽块级元素中，如果未为其指定修饰符 {{_}}（例如 {{@@[[div_]]@@}}）。_
该修饰符并非对所有块级元素都可用，详见各元素说明。

* 在简单引用块（{{>}}）中。

段落不会被创建：

* 在任何行内元素中。

* 在块级表格（{{@@[[table]]@@}}）中。

* 在大多数自由语法元素中（{{>}} 除外）。_
可以通过将所需文本包裹在 {{@@[[div]]@@}} 或 {{@@[[p]]@@}} 中来绕过该限制，例如：_
[[code]]|| [[div]]第一行

第二行[[/div]] || 下一个表格单元格 ||[[/code]]在此示例中，{{@@[[div]]@@}} 内的内容会被包裹为段落，而下一个表格单元格则会被直接作为文本添加。_
该技巧同样适用于块级表格。

要在支持段落的元素中创建或分隔新段落，需要满足以下多个条件：

* 该行必须是元素中的第一行，或者其上方至少有一整行空行，或者其上方存在一个全宽元素。

* 该行必须仅包含行内元素。全宽元素周围不会创建段落。在段落内部插入全宽元素会在该处终止当前段落，并在该全宽元素之后创建新段落。

如果在不创建段落的元素中存在文本（根据上述任一条件），文本中的空行将被视为普通换行（{{<br>}}，而不是 {{<p>}}）。

++ 控制换行

如果你希望空行仅作为空行，而不是创建段落，可以使用两种方法：

* 在该行放置任何视觉上为空的元素（但在段落判定上不视为空）。例如 {{@@[[span]][[/span]]@@}}、{{@<@>@@<@>@@<@>@@<@>@}}、{{@<&#64;&#60;>@@<&#62;&#64;>@}}。

* 在该行末尾添加符号 {{_}}。该符号会被明确解释为换行，并且绝不会转换为段落。

你也可以在代码分成多行时阻止换行（以及段落创建）。这在编写复杂代码时非常有用，可以保持可读性，同时不在视觉上拆分文本。为此，请在行末添加 {{\}}，则下一行会“粘连”到当前行。例如，下列代码在插入文章后将显示为一行 “abc”：

[[div class="code"]]
@@a\@@
@@[[span class="some-class"]]\@@
@@b\@@
@@[[/span]]\@@
@@c@@
[[/div]]

+ 自由语法

++ 文本格式

* {{@@**文本**@@}} —— **粗体** 文本。

* {{@@//文本//@@}} —— //斜体// 文本。

* {{@@{{文本}}@@}} —— 等宽文本。

* {{@@--文本--@@}} —— --删除线-- 文本。

* {{@@^^文本^^@@}} —— ^^上标^^ 文本。

* {{@@,,文本,,@@}} —— ,,下标,, 文本。

* {{@@__文本__@@}} —— __下划线__ 文本。

所有上述文本格式化方式都遵循相同的规则：

* 元素与其内部文本之间不得有空格（例如，{{@@__ 文本 __@@}} 不是正确语法）。

* 元素可以跨多行，但不能跨多个段落。
  正确：
[[code]]//a
b
c//[[/code]]
  错误：
[[code]]//a

b

c//[[/code]]

* 元素内部可以包含任何其他元素，包括块级元素。在使用块级元素的情况下，段落限制将被解除。例如：
[[code]]//[[div]]a

b

c[[/div]]//[[/code]]

同时也支持文本着色：

* {{@@##颜色|文本##@@}} — ##red|有颜色的## 文本。
  颜色可以使用任何 CSS 支持的颜色，例如表达式 {{@@##rgba(255, 127, 0, 0.5)|文本##@@}} 是合法的。
  当使用十六进制值表示颜色（RRGGBB、RRGGBBAA、RGB、RGBA）时，不必额外使用 {{#}} 符号（例如：{{@@##ffe|文本##@@}} 与 {{@@###ffe|文本##@@}} 等效）。

++ [[# links]] 链接

[[ul]]
[[li]]{{@@[地址 文本]@@}} 或 {{[*地址 文本]}} — 普通链接。
通常用于外部链接，或在站内文章之间使用参数进行链接，或用于锚点链接（例如：{{@@[#toc-0 指向第一个标题的链接]@@}}）。
地址不能包含空格，而链接文本可以包含空格。链接文本不得跨多行。
如果链接以星号 {{*}} 开头，则会在新窗口（标签页）中打开。
[[/li]]

[[li]]{{@@[[[文章]]]@@}} 或 {{@@[[[文章|]]]@@}} 或 {{@@[[[文章|链接文本]]]@@}} — 使用完整标识符（地址）创建站内文章链接。
此类链接会显示在文章的反向链接中；如果目标文章不存在，则会以不同颜色显示。

语法变体：

[[ul]]
  [[li]]{{@@[[[category:page1]]]@@}} — 创建指向 {{category:page1}} 的链接，链接文本大致对应文章标识符（符号 {{-}} 会被替换为空格，分类前缀会被移除等）。
  [[/li]]

  [[li]]{{@@[[[category:page1|]]]@@}} — 创建指向 {{category:page1}} 的链接。
  如果该文章存在，则自动使用其标题作为链接文本；否则使用其标识符。
  [[/li]]

  [[li]]{{@@[[[category:page1|文本]]]@@}} — 创建指向 {{category:page1}} 的链接，并使用指定文本作为链接名称。
  链接文本同样不得跨多行。
  [[/li]]
[[/ul]]
[[/li]]

[[li]]自动链接：形如 {{@@http://...@@}}、{{@@ftp://...@@}} 的文本会自动转换为对应链接。
这种自动替换可能会引发问题并破坏其他语法。如需避免，可以使用 [#literals 字面量]。
[[/li]]
[[/ul]]

++ 元素位置控制

* {{@@= 文本@@}} — 段落居中。必须放在段落开头使用，此后整个段落（包括后续行）都会居中对齐。

* {{@@_@@}} — 显式换行。只能用于行尾。
例如，可用于防止两个连续空行被合并为一个段落，如下所示：
[[code]]a
_
_
b[[/code]]

此外，{{@@_@@}} 允许在原本不支持换行的元素中插入换行（例如表格或列表）。
为了正确生效，显式换行符必须与前面的文本或元素之间用空格分隔。

* {{@@~~~@@}}、{{@@~~~<@@}}、{{@@~~~>@@}} — 清除浮动元素。
仅在行首使用时有效。
分别等同于：
{{@@[[div style="clear: both"]][/div]]@@}}
{{@@[[div style="clear: left"]][[/div]]@@}}
{{@@[[div style="clear: right"]][[/div]]@@}}

++ [[# headers]] 标题

{{@@+ 文本@@}}、
{{@@++ 文本@@}}、
{{@@+++ 文本@@}}、
{{@@++++ 文本@@}}、
{{@@+++++ 文本@@}}、
{{@@++++++ 文本@@}} — 分别对应一级至六级标题。

不支持六级以上标题。

该元素只能在新行使用。
内部不支持换行，但可以嵌入块级元素或显式换行，例如：

[[code]]++ 第一行 _
第二行[[/code]]

在标题文本前可以添加符号 {{*}}，例如 {{@@++* 文本@@}}。
此时该标题不会被加入目录。

++ 水平线

{{@@---@@}} — 添加水平分隔线。

该元素只能在新行使用。
前三个 {{-}} 之后的数量没有限制。

++ 引用

以下语法会创建两个嵌套的引用块：

[[code]]
> 第一级
>> 第二级
>> 第二级的第二行
> 回到第一级
[[/code]]

由于 {{@@[[blockquote]]@@}} 具有更好的可读性（和可编辑性），因此不推荐使用此语法。

++ 列表

网站支持三种列表类型：有序列表、无序列表和字典列表。
前两种可以相互嵌套。

示例：无序列表中嵌套有序列表：

[[code]]
* 元素1
* 元素2
 # 元素2.1
 # 元素2.2
* 元素3
[[/code]]

列表的嵌套级别由 {{*}} 或 {{#}} 前的空格或制表符数量决定。

列表项中可以通过 {{_}} 或使用 {{@@[[span]]@@}} 包裹内容来实现多行，例如：

[[code]]
* 元素1 _
下一行
* [[span]]元素2
下一行[[/span]]
[[/code]]

字典列表定义如下：

[[code]]
: 术语1 : 定义
: 术语2 : 定义
: 术语3 : 定义
[[/code]]

在术语或定义中同样可以使用 {{_}} 或 {{@@[[span]]@@}}。

++ 表格

简化（自由）语法的表格如下所示：

[[code]]
||~ 标题 ||~ 标题2 ||
||> 右对齐文本 ||= 居中文本 ||
|||| 横向跨越两列的单元格 ||
[[/code]]

要在单元格中添加多行文本，可以使用 {{_}} 或 {{@@[[span]]@@}}。

该元素无法创建跨越多行（纵向合并）的单元格。在这种情况下，可以使用块级元素 {{@@[[table]]@@}}，它不受此限制。

++ 其他

* {{@@[[# anchor]]@@}} — 创建一个具有指定标识符的元素，从而可以通过链接跳转到该位置（例如：{{@@[#anchor 跳转到锚点]@@}}）。
  该元素在视觉上类似于块级元素，但实际上并不是。
  在元素定义中，{{#}} 后必须保留一个空格。

* {{@@[!-- 注释 --]@@}} — 定义一段在最终页面显示时不会呈现的源代码区域。
  可用于在代码中添加技术性备注。
  [!-- 我就知道你会看到这里。 --]

* [[# literals]]{{@<&#64;&#64;>@文本@<&#64;&#64;>@}} — 阻止 {{@<&#64;&#64;>@}} 包裹的内容被当作标记语法解析。
  始终生成一行文本（字面量）。
  可用于在出于视觉效果使用标记符号时避免歧义（例如，并非链接用途的单个方括号）。
  也可用于破坏自动替换语法（当不希望发生自动替换时），例如：
  {{%%pat@<&#64;&#64;>@h|param%%}} 始终会显示为文本 %%pat@@@@h|param%%，即便指定了参数 {{param}}。
  也可用于像 {{_}} 那样创建空行（但不推荐这样使用）。

* {{@<&#64;&#60;>@&mdash;&copy;@<&#62;&#64;>@}} — 插入指定的 HTML 实体符号（或多个符号）。

* 符号 «、» 和 — 的替换
_
_
由于这些替换不属于自动替换，因此在仅支持纯文本的场景下不会生效（例如块属性或链接名称中）。

 * {{@@ <<@@}} — 左引号：«

 * {{@@ >>@@}} — 右引号：»

 * {{@@ --@@}} — 长破折号：—
   需要注意的是，为了使该元素被解析为破折号而不是删除线语法，其两侧必须有空格。

+ 块级元素

所有块级元素都遵循类似规则构建。

每个块级元素都有名称（例如 {{div}}、{{iftags}}、{{blockquote}} 等）、可选的标识符，以及一个起始标签（例如 {{@@[[div]]@@}}）。
可以包含文本或其他块级元素的块级元素，还必须有与起始标签对应名称的结束标签（例如 {{@@[[/div]]@@}}）。

某些块可以使用修饰符 {{_}}。
该修饰符写在块名称之后，用于阻止在块内自动创建段落（文本会直接放入块中，换行通过 {{<br>}} 实现）。
修饰符只写在起始标签中，因此以下语法是正确的：
[[code]][[div_]]text[[/div]][[/code]]

块的标识符是可选文本，写在块名称之后（但在修饰符 {{_}} 之前），通过 {{:}} 指定。例如：

[[code]]
[[module:lu ListUsers]]
  [[module CSS]]
    body {
      background: url(%%avatar%%);
    }
  [[/module]]
[[/module:lu]]
[[/code]]

块标识符允许在接受文本内容的块（如 {{@@[[code]]@@}}、{{@@[[module]]@@}}、{{@@[[html]]@@}}）内部使用该块的标准结束标签，而不会真正关闭它。
这样可以将多个模块相互嵌套，或在该块内部写出 {{@@[[code]]@@}} 的示例：

[[code:outer]]
[[code:2]]
  [[code:b]]
    示例：使用块 [[cоde]]
  [[/code:b]]
[[/code:2]]
[[/code:outer]]

大多数块可以以某种形式接受属性（要么是 HTML 属性，要么是特定块自定义属性）。
属性写法为 {{参数=值}} 或 {{参数="值"}}。
不同于 HTML，在属性值中使用特殊字符时，不使用 HTML 实体（如 {{&quot;}}），而是通过 {{\}} 转义：
例如 {{@@[[collapsible show="协议 \"忧郁\""]]@@}}。

++ [[# html-attributes]] 标准 HTML 属性

某些元素（例如 {{@@[[a]]@@}}、{{@@[[span]]@@}} 等）是 HTML 的直接接口，
其标记中的属性会直接插入生成的 HTML 页面中。

并非所有 HTML 属性都允许在标记中使用。
允许使用的属性列表如下；更多详情请参阅
https://www.w3schools.com/tags/ref_attributes.asp 的 HTML 文档。
在本网站语境中最常用的属性已用 ##red|红色## 标出。

* {{##red|alt##}}
* {{##red|class##}}
* {{##red|colspan##}}
* {{##red|href##}}
* {{##red|id##}}：该属性会被特殊处理。标识符必须以 {{u-}} 作为前缀。如果未指定此前缀，系统会自动添加。例如，{{id="myid"}} 将会被转换为 {{id="u-myid"}}。
* {{##red|rowspan##}}
* {{##red|style##}}
* {{##red|target##}}
* {{accept}}
* {{align}}
* {{autocapitalize}}
* {{autoplay}}
* {{background}}
* {{bgcolor}}
* {{border}}
* {{buffered}}
* {{checked}}
* {{cite}}
* {{cols}}
* {{contenteditable}}
* {{controls}}
* {{coords}}
* {{datetime}}
* {{decoding}}
* {{default}}
* {{dir}}
* {{dirname}}
* {{disabled}}
* {{download}}
* {{draggable}}
* {{for}}
* {{form}}
* {{headers}}
* {{height}}
* {{hidden}}
* {{high}}
* {{hreflang}}
* {{inputmode}}
* {{ismap}}
* {{itemprop}}
* {{kind}}
* {{label}}
* {{lang}}
* {{list}}
* {{loop}}
* {{low}}
* {{max}}
* {{maxlength}}
* {{min}}
* {{minlength}}
* {{multiple}}
* {{muted}}
* {{name}}
* {{optimum}}
* {{pattern}}
* {{placeholder}}
* {{poster}}
* {{preload}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{role}}
* {{rows}}
* {{scope}}
* {{selected}}
* {{shape}}
* {{size}}
* {{sizes}}
* {{span}}
* {{spellcheck}}
* {{src}}
* {{srclang}}
* {{srcset}}
* {{start}}
* {{step}}
* {{tabindex}}
* {{title}}
* {{translate}}
* {{type}}
* {{usemap}}
* {{value}}
* {{width}}
* {{wrap}}
* {{scrolling}}
* {{frameborder}}

++ [[# booleans]] 布尔属性

文档中标记为可接受布尔值的属性，在实际使用中可以用以下字符串表示：

True：

* {{true}}
* {{t}}
* {{1}}
* {{yes}}

False：

* {{false}}
* {{f}}
* {{0}}
* {{no}}

同样适用于以下内置 HTML 属性：

* {{allowfullscreen}}
* {{allowpaymentrequest}}
* {{async}}
* {{autofocus}}
* {{autoplay}}
* {{checked}}
* {{controls}}
* {{default}}
* {{disabled}}
* {{formnovalidate}}
* {{hidden}}
* {{ismap}}
* {{itemscope}}
* {{loop}}
* {{multiple}}
* {{muted}}
* {{nomodule}}
* {{novalidate}}
* {{open}}
* {{playsinline}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{selected}}
* {{truespeed}}

++ {{@@[[<]]@@}}、{{@@[[>]]@@}}、{{@@[[=]]@@}}、{{@@[[==]]@@}}：对齐

: 类型 : 全宽
: 段落 : ✅
: 支持属性 : ❌

* {{@@[[<]]@@}} — 将内部文本左对齐。

* {{@@[[>]]@@}} — 将内部文本右对齐。

* {{@@[[=]]@@}} — 将内部文本居中对齐。

* {{@@[[==]]@@}} — 将内部文本两端对齐。

++ {{@@[[a]]@@}}：链接

: 类型 : 行内
: 别名 : {{@@[[anchor]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{*}} : ✅
: 支持 {{_}} : ✅

使用修饰符 {{*}}（例如 {{@@[[*a href="https://google.com"]]Google[[/a]]@@}}）等同于使用 {{target="_blank"}}；该链接会在新窗口（标签页）中打开。
若同时使用该修饰符和 {{target}}，其值将会叠加。

通过该元素创建的链接会经过 [#link-handling 标准过滤]。

++ {{@@[[blockquote]]@@}}：引用块

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[quote]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

在功能上等同于使用 {{>}}，但在标记层面更加“干净”和易读。

++ {{@@[[b]]@@}}：加粗文本

: 类型 : 行内
: 别名 : {{@@[[bold]]@@}}、{{@@[[strong]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[char]]@@}}：HTML 字符

: 类型 : 行内
: 别名 : {{@@[[character]]@@}}
: 支持属性 : ❌

在文本中插入一个 HTML 实体字符。
其工作方式与 {{@<&#64;&#60;>@@<&#62;&#64;>@}} 语法几乎相同。

使用示例：{{@@[[char &mdash;]]@@}}

++ {{@@[[code]]@@}}：代码块

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许忽略 {{@@[[code]]@@}} 与 {{@@[[/code]]@@}} 之间的标记规则。
主要用于展示标记示例而不被立即解析。

该元素也可用于代码高亮。

通过该元素添加到页面的代码，可以通过如下格式的单独文件访问：

{{@@https@@://files.projwikit.unitreaty.org/local@@--@@code/<页面名称>/<页面中的代码块编号，从 1 开始>}}

对于 HTML、JavaScript、XML、CSS 语言，将设置对应的 MIME 类型，
从而可以在 {{<script src>}}、{{<link rel>}} 等需要类型匹配的场景中使用。

仅支持一个属性：

* {{type}} — 需要高亮的编程语言名称。

[[collapsible show="[+] 支持语言完整列表" hide="[-] 支持语言完整列表"]]

||~ 语言名称                ||~ 缩写                ||
|| 1C                      || 1c                     ||
|| ABNF                    || abnf                   ||
|| Access logs             || accesslog              ||
|| Ada                     || ada                    ||
|| Arduino (C++ w/Arduino libs) || arduino, ino           ||
|| ARM assembler           || armasm, arm            ||
|| AVR assembler           || avrasm                 ||
|| ActionScript            || actionscript, as       ||
|| AngelScript             || angelscript, asc       ||
|| Apache                  || apache, apacheconf     ||
|| AppleScript             || applescript, osascript ||
|| Arcade                  || arcade                 ||
|| AsciiDoc                || asciidoc, adoc         ||
|| AspectJ                 || aspectj                ||
|| AutoHotkey              || autohotkey             ||
|| AutoIt                  || autoit                 ||
|| Awk                     || awk, mawk, nawk, gawk  ||
|| Bash                    || bash, sh, zsh          ||
|| Basic                   || basic                  ||
|| BNF                     || bnf                    ||
|| Brainfuck               || brainfuck, bf          ||
|| C#                      || csharp, cs             ||
|| C                       || c, h                   ||
|| C++                     || cpp, hpp, cc, hh, c++, h++, cxx, hxx ||
|| C/AL                    || cal                    ||
|| Cache Object Script    || cos, cls               ||
|| CMake                   || cmake, cmake.in        ||
|| Coq                     || coq                    ||
|| CSP                     || csp                    ||
|| CSS                     || css                    ||
|| Cap’n Proto             || capnproto, capnp       ||
|| Clojure                 || clojure, clj           ||
|| CoffeeScript            || coffeescript, coffee, cson, iced ||
|| Crmsh                   || crmsh, crm, pcmk       ||
|| Crystal                 || crystal, cr            ||
|| D                       || d                      ||
|| Dart                    || dart                   ||
|| Delphi                  || dpr, dfm, pas, pascal  ||
|| Diff                    || diff, patch            ||
|| Django                  || django, jinja          ||
|| DNS Zone file           || dns, zone, bind        ||
|| Dockerfile              || dockerfile, docker     ||
|| DOS                     || dos, bat, cmd          ||
|| dsconfig                || dsconfig               ||
|| DTS (Device Tree)       || dts                    ||
|| Dust                    || dust, dst              ||
|| EBNF                    || ebnf                   ||
|| Elixir                  || elixir                 ||
|| Elm                     || elm                    ||
|| Erlang                  || erlang, erl            ||
|| Excel                   || excel, xls, xlsx       ||
|| F#                      || fsharp, fs             ||
|| FIX                     || fix                    ||
|| Fortran                 || fortran, f90, f95      ||
|| G-Code                  || gcode, nc              ||
|| Gams                    || gams, gms              ||
|| GAUSS                   || gauss, gss             ||
|| Gherkin                 || gherkin                ||
|| Go                      || go, golang             ||
|| Golo                    || golo, gololang         ||
|| Gradle                  || gradle                 ||
|| GraphQL                 || graphql                ||
|| Groovy                  || groovy                 ||
|| HTML, XML               || xml, html, xhtml, rss, atom, xjb, xsd, xsl, plist, svg ||
|| HTTP                    || http, https            ||
|| Haml                    || haml                   ||
|| Handlebars              || handlebars, hbs, html.hbs, html.handlebars        ||
|| Haskell                 || haskell, hs            ||
|| Haxe                    || haxe, hx               ||
|| Hy                      || hy, hylang             ||
|| Ini, TOML               || ini, toml              ||
|| Inform7                 || inform7, i7            ||
|| IRPF90                  || irpf90                 ||
|| JSON                    || json                   ||
|| Java                    || java, jsp              ||
|| JavaScript              || javascript, js, jsx    ||
|| Julia                   || julia, julia-repl      ||
|| Kotlin                  || kotlin, kt             ||
|| LaTeX                   || tex                    ||
|| Leaf                    || leaf                   ||
|| Lasso                   || lasso, ls, lassoscript ||
|| Less                    || less                   ||
|| LDIF                    || ldif                   ||
|| Lisp                    || lisp                   ||
|| LiveCode Server         || livecodeserver         ||
|| LiveScript              || livescript, ls         ||
|| Lua                     || lua                    ||
|| Makefile                || makefile, mk, mak, make ||
|| Markdown                || markdown, md, mkdown, mkd ||
|| Mathematica             || mathematica, mma, wl   ||
|| Matlab                  || matlab                 ||
|| Maxima                  || maxima                 ||
|| Maya Embedded Language  || mel                    ||
|| Mercury                 || mercury                ||
|| Mizar                   || mizar                  ||
|| Mojolicious             || mojolicious            ||
|| Monkey                  || monkey                 ||
|| Moonscript              || moonscript, moon       ||
|| N1QL                    || n1ql                   ||
|| NSIS                    || nsis                   ||
|| Nginx                   || nginx, nginxconf       ||
|| Nim                     || nim, nimrod            ||
|| Nix                     || nix                    ||
|| OCaml                   || ocaml, ml              ||
|| Objective C             || objectivec, mm, objc, obj-c, obj-c++, objective-c++ ||
|| OpenGL Shading Language || glsl                   ||
|| OpenSCAD                || openscad, scad         ||
|| Oracle Rules Language   || ruleslanguage          ||
|| Oxygene                 || oxygene                ||
|| PF                      || pf, pf.conf            ||
|| PHP                     || php                    ||
|| Parser3                 || parser3                ||
|| Perl                    || perl, pl, pm           ||
|| Plaintext               || plaintext, txt, text   ||
|| Pony                    || pony                   ||
|| PostgreSQL & PL/pgSQL   || pgsql, postgres, postgresql ||
|| PowerShell              || powershell, ps, ps1    ||
|| Processing              || processing             ||
|| Prolog                  || prolog                 ||
|| Properties              || properties             ||
|| Protocol Buffers        || protobuf               ||
|| Puppet                  || puppet, pp             ||
|| Python                  || python, py, gyp        ||
|| Python profiler results || profile                ||
|| Python REPL             || python-repl, pycon     ||
|| Q                       || k, kdb                 ||
|| QML                     || qml                    ||
|| R                       || r                      ||
|| ReasonML                || reasonml, re           ||
|| RenderMan RIB           || rib                    ||
|| RenderMan RSL           || rsl                    ||
|| Roboconf                || graph, instances       ||
|| Ruby                    || ruby, rb, gemspec, podspec, thor, irb ||
|| Rust                    || rust, rs               ||
|| SAS                     || SAS, sas               ||
|| SCSS                    || scss                   ||
|| SQL                     || sql                    ||
|| STEP Part 21            || p21, step, stp         ||
|| Scala                   || scala                  ||
|| Scheme                  || scheme                 ||
|| Scilab                  || scilab, sci            ||
|| Shell                   || shell, console         ||
|| Smali                   || smali                  ||
|| Smalltalk               || smalltalk, st          ||
|| SML                     || sml, ml                ||
|| Stan                    || stan, stanfuncs        ||
|| Stata                   || stata                  ||
|| Stylus                  || stylus, styl           ||
|| SubUnit                 || subunit                ||
|| Swift                   || swift                  ||
|| Tcl                     || tcl, tk                ||
|| Test Anything Protocol  || tap                    ||
|| Thrift                  || thrift                 ||
|| TP                      || tp                     ||
|| Twig                    || twig, craftcms         ||
|| TypeScript              || typescript, ts         ||
|| VB.Net                  || vbnet, vb              ||
|| VBScript                || vbscript, vbs          ||
|| VHDL                    || vhdl                   ||
|| Vala                    || vala                   ||
|| Verilog                 || verilog, v             ||
|| Vim Script              || vim                    ||
|| X++                     || axapta, x++            ||
|| x86 Assembly            || x86asm                 ||
|| XL                      || xl, tao                ||
|| XQuery                  || xquery, xpath, xq      ||
|| YAML                    || yml, yaml              ||
|| Zephir                  || zephir, zep            ||

[[/collapsible]]

++ {{@@[[collapsible]]@@}}：可折叠区块

: 类型 : 全宽
: 段落 : ✅
: 支持 HTML 属性 : ❌

创建一个可通过按钮展开或折叠的区块。

除 HTML 属性外，还支持以下属性：

* {{show}} — 区块关闭时按钮显示的文本。

* {{hide}} — 区块打开时按钮显示的文本。

* {{align}} — 按钮文本对齐方式。可选值：
  {{left}}、{{right}}、{{center}}、{{justify}}。

* {{folded}} — 指示区块是否默认展开。[#booleans 布尔值]。

* {{hideLocation}} — 指示展开后关闭按钮的显示位置。
  可选值：{{top}}、{{bottom}}、{{both}}、{{neither}}、{{none}}。
  后两个选项等价。

++ [[# date]] {{@@[[date]]@@}}：日期

: 类型 : 行内
: 支持 HTML 属性 : ❌

插入日期，并根据查看者计算机的时区自动调整显示。

使用特殊属性语法；日期直接写在块名后，而非单独属性：

[[code]]
[[date 2024-02-18T00:00:00Z format="%H:%M:%S %d.%m.%Y"]]
[[/code]]

例如，该日期以 UTC 指定，
但在 UTC+0200 时区的计算机上将显示为：

"02:00:00 18.02.2024"

++ {{@@[[div]]@@}}：全宽通用容器

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

++ [[# footnotes]] {{@@[[footnote]]@@}}：脚注

: 类型 : 行内
: 支持属性 : ❌

在文本中添加编号脚注。
其内容默认显示在页面底部，或显示在 {{@@[[footnoteblock]]@@}} 所在位置（若手动指定）。

++ {{@@[[footnoteblock]]@@}}：脚注块

: 类型 : 全宽
: 支持 HTML 属性 : ❌

显示文章中所有脚注的列表。

支持以下属性：

* {{title}} — 显示在脚注列表上方的标题文本。

* {{hide}} — [#booleans 布尔值]。
  若指定，该脚注块将不显示。
  可用于移除页面自动添加的默认脚注块，使脚注仅以悬浮提示形式显示。

++ {{@@[[form]]@@}}：表单

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

允许填写表单并通过 [#path-params URL 参数] 提交到站内指定页面。

支持所有常规 HTML 表单属性，
但 {{target}} 属性具有特殊含义：它应填写完整的文章标识符，而不是 URL。

使用示例：

[[code]]
[[form target="search"]]
  [[input type="text" name="s"]]
  [[input type="submit"]]
[[/form]]
[[/code]]

点击“提交”后，将跳转到文章 {{search}}，
例如：

{{@@https://projwikit.unitreaty.org/search/s/你的_字符串@@}}

随后可在目标文章中使用 {{@@%%path%%@@}} 功能读取该参数。

++ {{@@[[html]]@@}}: HTML 代码

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许插入任意 HTML 代码，系统会自动将其包裹在 {{@@[[iframe]]@@}} 中。以这种方式创建的框架会自动根据其内容大小进行自适应。

在当前版本的网站中，该元素通过 {{<iframe srcdoc="...">}} 实现。通过这种方式创建的框架没有域名，因此在框架内部的 JS 代码中无法使用 {{window.localStorage}} 和 {{document.cookie}}。

支持以下属性：

* {{external}} — [#booleans 布尔值]（默认 — {{false}}）；启用该参数后，区块将通过媒体域名（https://files.projwikit.unitreaty.org）进行渲染，其方式与 Wikidot 平台相同。此类区块通常加载时间会稍慢几秒，但支持使用 {{window.localStorage}} 和 {{document.cookie}}。
**注意：** 当前页面版本系统在启用 {{external}} 的 HTML 区块中无法正常工作，因此始终会显示你代码的最新版本（包括在预览和查看旧版本页面时）。此外，此类区块不会读取 [#path-params 页面参数]。

++ {{@@[[iframe]]@@}}: 通过链接嵌入外部页面

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

语法：

[[code]]
[[iframe 链接 属性1="值" 属性2="值"]]
[[/code]]

支持所有允许的 HTML 属性。

通过该元素插入的链接会经过 [#link-handling 标准过滤]。

++ [[# if]] {{@@[[#if]]@@}}, {{@@[[if]]@@}}: 根据值是否存在显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

行内示例：

[[code]]
[[#if {$param} | param exists | param does not exist]]
[[/code]]

全宽示例：

[[code]]
[[if {$param}]]
  param exists
[[else]]
  param does not exist
[[/if]]
[[/code]]

只要是非空字符串都被视为“存在”，//除了// {{@@{$变量}@@}} 或 {{@@%%变量%%@@}} 这种格式的字符串。由于未在 {{@@[[include]]@@}} 或模块中传递的参数会保留为文本值，因此可以借此判断参数是否被传入。

++ [[# ifexpr]] {{@@[[#ifexpr]]@@}}, {{@@[[ifexpr]]@@}}: 根据条件表达式显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

块级示例：

[[code]]
[[module CountPages fullname="main"]]
  [[#ifexpr %%count%% > 0 | yes | no]]
[[/module]]
[[/code]]

全宽示例：

[[code]]
[[module CountPages fullname="main"]]
  [[ifexpr %%count%% > 0]]
    main page exists
  [[else]]
    main page does not exist
  [[/ifexpr]]
[[/module]]
[[/code]]

若表达式语法错误（例如变量不存在或参数传递错误），表达式视为 false。

否则，除 {{False}}、{{0}}、{{0.0}} 或 {{""}} 以外的任何值都视为 true。

支持的运算符：{{*}}, {{/}}, {{+}}, {{-}}, {{@@ <<@@}}, {{@@>>@@}}, 以及比较运算 {{==}}, {{!=}}, {{<}}, {{>}}, {{<=}}, {{>=}}。

除直接比较外，大多数运算仅支持数字。字符串必须以 JSON 格式表示（例如 {{"值"}}）。

还支持以下函数：

* {{min(x, y, ...)}} — 返回最小值。
* {{max(x, y, ...)}} — 返回最大值。
* {{abs(x)}} — 返回绝对值。
* {{round(x)}} — 四舍五入为整数。
* {{lower(str)}} — 转为小写。
* {{upper(str)}} — 转为大写。

++ [[# ifcategory]] {{@@[[ifcategory]]@@}}: 根据文章分类显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

示例：

[[code]]
[[ifcategory +theme]]
该文本仅在 theme 分类页面中显示。
[[/ifcategory]]
[[ifcategory -theme]]
该文本在除 theme 外的所有页面显示。
[[/ifcategory]]
[[/code]]

该元素主要用于与 {{[[include]]}} 配合使用，使插入的代码根据所在页面的分类产生不同效果。

若存在多个分类，应以空格分隔。

++ [[# iftags]] {{@@[[iftags]]@@}}: 根据文章标签显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

标签条件写在区块名称后，以空格分隔，格式如下：

* {{+标签}} — 页面必须包含该标签。
* {{-标签}} — 页面不得包含该标签。
* {{标签}} — 页面至少包含列出的其中一个标签。

示例：

[[code]]
[[iftags 对象 故事 +ru]]
该文章属于俄罗斯分部的对象或故事。
[[/iftags]]
[[/code]]

++ {{@@[[image]]@@}}: 图片

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅

插入当前页面附件图片：

[[code]]
[[image img.png alt="my img"]]
[[/code]]

插入其他页面附件图片：

[[code]]
[[image main/ico_arthub.svg alt="image from another page"]]
[[/code]]

也可以插入互联网图片（不推荐，因为可能发生链接失效——上传至站点的文件会随文章长期保留，而网络图片可能随时失效）。

支持对齐修饰符 {{f<}}, {{f>}}, {{<}}, {{>}}, {{=}}。后三种会使图片成为全宽元素。

* {{@@[[f<image]]@@}} — 左浮动图片，文字会环绕。
* {{@@[[f>image]]@@}} — 右浮动图片。
* {{@@[[<image]]@@}} — 左对齐全宽图片。
* {{@@[[>image]]@@}} — 右对齐全宽图片。
* {{@@[[=image]]@@}} — 居中全宽图片。

支持 {{link}} 属性，可将图片自动包裹为链接。该链接会经过 [#link-handling 标准过滤]，效果等同于 {{@@[[a href="..."]][[image ...]][[/a]]@@}}。

++ {{@@[[input]]@@}}: 输入框

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅

可单独使用，也可与表单搭配使用。

++ {{@@[[i]]@@}}: 斜体文本

: 类型 : 行内
: 别名 : {{@@[[italics]]@@}}, {{@@[[em]]@@}}, {{@@[[emphasis]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[lines]]@@}}: 插入空行

: 类型 : 行内
: 支持属性 : ❌

用于插入多行空白。

示例：

[[code]]
[[lines 8]]
[[/code]]

++ {{@@[[ul]]@@}}, {{@@[[ol]]@@}}, {{@@[[li]]@@}}: 列表

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

* {{@@[[ul]]@@}} — 无序列表（全宽）。
* {{@@[[ol]]@@}} — 有序列表（全宽）。
* {{@@[[li]]@@}} — 列表项（行内）。

示例：

[[code]]
[[ul]]
  [[li]]第一项[[/li]]
  [[li]]第二项[[/li]]
  [[li]]
    [[ol]]
      [[li]]第一条编号[[/li]]
      [[li]]第二条编号[[/li]]
    [[/ol]]
  [[/li]]
  [[li]]第三项[[/li]]
[[/ul]]
[[/code]]

所有列表元素均支持标准 HTML 属性。

++ {{@@[[mark]]@@}}: 高亮文本

: 类型 : 行内
: 别名 : {{@@[[highlight]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# module-block]] {{@@[[module]]@@}}: 插入站点模块

: 类型 : 全宽
: 别名 : {{@@[[module654]]@@}}
: 支持 HTML 属性 : ❌

通用语法：

[[code]]
[[module 模块名称 属性1="值" 属性2="值"]]
模块内容（仅适用于支持内容的模块）
[[/module]]
[[/code]]

模块列表及其属性见“模块”章节。

对于不支持内容的模块（如 {{@@[[module Rate]]@@}}），无需写关闭标签。

++ {{@@[[table]]@@}}, {{@@[[row]]@@}}, {{@@[[hcell]]@@}}, {{@@[[cell]]@@}}: 表格

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

类似于标准 HTML 表格。

* {{@@[[table]]@@}} — {{<table>}}
* {{@@[[row]]@@}} — {{<tr>}}
* {{@@[[hcell]]@@}} — {{<th>}}
* {{@@[[cell]]@@}} — {{<td>}}

不支持 {{<thead>}}、{{<tbody>}}、{{<tfoot>}}。

示例：

[[code]]
[[table class="wiki-content-table"]]
  [[row]]
    [[hcell]]标题 1[[/hcell]]
    [[hcell]]标题 2[[/hcell]]
    [[cell rowspan="2"]]跨行单元格[[/cell]]
  [[/row]]
  [[row]]
    [[cell colspan="2"]]跨列单元格[[/cell]]
  [[/row]]
[[/table]]
[[/code]]

++ {{@@[[tabview]]@@}}, {{@@[[tab]]@@}}: 标签页

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[tabs]]@@}}（对应 {{@@[[tabview]]@@}}）
: 支持 HTML 属性 : ❌

用于在页面中创建可切换的标签页。

示例：

[[code]]
[[tabview]]
  [[tab 标签 1]]
    标签 1 内容
  [[/tab]]
  [[tab 标签 2]]
    标签 2 内容
  [[/tab]]
[[/tabview]]
[[/code]]

标签名称也可使用 {{@@[[tab title="标签名称"]]@@}} 格式。

++ {{@@[[tt]]@@}}: 等宽文本

: 类型 : 行内
: 别名 : {{@@[[mono]]@@}}, {{@@[[monospace]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[p]]@@}}：显式段落

: 类型 : 全宽
: 替代名称 : {{@@[[paragraph]]@@}}
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

允许为当前段落指定 HTML 属性，就像为 {{<p>}} 指定属性一样。

示例：

[[code]]
[[p style="color: red"]]
红色段落。
[[/p]]
[[/code]]

++ {{@@[[ruby]]@@}}, {{@@[[rt]]@@}}：汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[rubytext]]@@}}（用于 {{@@[[rt]]@@}}）
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[ruby]]マレニア[[rt]]Malenia[[/rt]][[/ruby]]
[[/code]]

++ {{@@[[rb]]@@}}：简化汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[ruby2]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[rb マレニア | Malenia]]
[[/code]]

++ {{@@[[size]]@@}}：字体大小

: 类型 : 行内
: 支持属性 : ❌

允许修改字体大小。尺寸可以使用任何 CSS 允许的数值。

~~~

使用示例：

[[code]]
这段文字非常[[size 200%]]大[[/size]]，同时又[[size 6px]]小[[/size]]。
[[/code]]

++ {{@@[[span]]@@}}：行内通用容器

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

可以包含任何其他元素，也可用于为文章中的文本添加样式。

++ {{@@[[s]]@@}}：删除线文本

: 类型 : 行内
: 替代名称 : {{@@[[strikethrough]]@@}}, {{@@[[del]]@@}}, {{@@[[deletion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sup]]@@}}：上标文本

: 类型 : 行内
: 替代名称 : {{@@[[super]]@@}}, {{@@[[superscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sub]]@@}}：下标文本

: 类型 : 行内
: 替代名称 : {{@@[[subscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[toc]]@@}}：自动目录

: 类型 : 全宽
: 支持属性 : ❌

在页面中添加一个可展开的标题列表块。

可使用以下前缀改变元素位置：

* {{@@[[f<toc]]@@}} — 左侧浮动块。该块会被周围文本和块级元素环绕。

* {{@@[[f>toc]]@@}} — 右侧浮动块。

++ {{@@[[u]]@@}}：下划线文本

: 类型 : 行内
: 替代名称 : {{@@[[underline]]@@}}, {{@@[[ins]]@@}}, {{@@[[insertion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# user-block]] {{@@[[user]]@@}}：用户链接

: 类型 : 行内
: 支持属性 : ❌
: 支持 {{*}} : ✅

可以以两种形式使用：

* {{@@[[user 用户名]]@@}} — 输出普通用户链接。

* {{@@[[*user 用户名]]@@}} — 输出用户链接及其头像。

+ [[# link-handling]] 补充：链接处理

出于安全原因，某些链接会被网站屏蔽。

除非某个元素或模块的文档中另有说明，否则所有链接（例如 {{@@[]@@}}、{{@@[[[|]]]@@}}、{{@@[[a href="..."]]@@}}、{{@@[[image ... link="..."]]@@}} 等）都会按照以下规则进行检查。

明确 **禁止** 的绝对链接协议：

* {{data:}}
* {{javascript:}}（除 {{"javascript:;"}} 之外，该形式表示“空链接”）

明确允许的绝对链接协议：

* {{blob:}}
* {{@@chrome-extension://@@}}
* {{@@chrome://@@}}
* {{@@content://@@}}
* {{data:}}
* {{dns:}}
* {{feed:}}
* {{@@file://@@}}
* {{@@ftp://@@}}
* {{@@git://@@}}
* {{@@gopher://@@}}
* {{@@http://@@}}
* {{@@https://@@}}
* {{@@irc6://@@}}
* {{@@irc://@@}}
* {{@@ircs://@@}}
* {{mailto:}}
* {{@@resource://@@}}
* {{@@rtmp://@@}}
* {{@@sftp://@@}}

对于块级元素，下列以指定字符开头的链接同样被视为合法链接：

* {{A-Z}}, {{a-z}}, {{0-9}}, {{.}}（普通相对链接）。
* {{/}}, {{@@//@@}}（基于当前域名和协议的绝对链接）。
* {{#}}（锚点跳转）。
* {{?}}（跳转到当前页面并附带 GET 参数）。
* {{$}}, {{&}}, {{+}}, {{,}}, {{:}}, {{;}}, {{=}}, {{@}}, {{%}}, {{-}}, {{~}}（其他允许用于相对链接的特殊字符）。

对于自由语法元素，会采用更严格的校验规则，并允许更少的特殊字符，以便将链接与语法本身区分开来：

* {{#}} + ({{A-Z}}, {{a-z}}, {{0-9}}, {{_}}, {{-}}, {{%}})（锚点跳转）。
* {{@@//地址@@}}（使用当前协议的绝对链接）。
* {{/地址}}（使用当前域名的绝对链接）。
* 任何不以 {{/}} 开头，但包含它的文本。

+ 补充：模块

模块提供了网站中不属于普通文章的全部功能。这包括交互元素（评分、论坛、标签云）以及扩展功能（向文章添加 CSS 样式、获取其他文章或当前用户的信息等）。

如果模块的运行导致网站功能异常，可以通过添加参数 {{/nomodule/true}} 访问页面（例如：{{@@https://projwikit.unitreaty.org/main/nomodule/true@@}}）。
该参数会完全禁用此页面上的 **所有** 模块。

关于如何在文章中插入模块的详细说明，请参见 [#module-block 章节] {{@@[[module]]@@}}。

此外，需要注意的是，对于支持内部标记的模块（例如 {{ListUsers}}、{{ListPages}} 和 {{CountPages}}），模块内容在技术上并不属于文章内容的一部分。例如，在模块内部定义的任何 [#headers 标题] 或 [#footnotes 脚注]，都只会在该模块范围内显示。

++ 模块 Rate

: 支持参数 : ❌
: 支持内容 : ❌

为文章添加评分组件，效果类似于页面底部“评分”按钮下方的评分模块。

从发布规则角度来看，该模块是必需的，但从技术角度来看并非强制。如果文章中未包含该模块，仍然可以通过“评分”按钮进行投票。

该模块不接受任何参数。

++ 模块 CSS

: 支持参数 : ❌
: 支持内容 : ✅

该模块的内容会原样作为 CSS 样式应用到当前页面。例如：

[[code]]
[[module CSS]]
body {
  background: red;
}
[[/module]]
[[/code]]

可以通过如下地址获取页面上所有 CSS 模块合并后的结果文件：
{{@@https@@://files.projwikit.unitreaty.org/local@@--@@theme/<页面名称>/style.css}}

该方法会处理 {{@@[[if]]@@}}、{{@@[[ifexpr]]@@}} 和 {{@@[[noinclude]]@@}}。
可以通过参数 {{?includeParams=<JSON 格式参数>}} 传递参数。

++ [[# module-listpages]] 模块 ListPages

: 支持参数 : ✅
: 支持内容 : ✅

用于根据指定参数获取文章列表。

对于每一篇找到的文章，模块内容都会被复制并作为标记进行处理，并进行 [#autoreplace 自动替换]，替换以下变量：

* {{%%name%%}} — 文章的自身标识符（例如 {{main}}）。
* {{%%category%%}} — 文章类别（例如 {{sandbox}}）。
* {{%%fullname%%}} — 包含类别的完整标识符（例如 {{sandbox:main}}）。
* {{%%title%%}} — 文章标题。
* {{%%title_linked%%}} — 包含标题的 [#links 内部链接]。
* {{%%link%%}} — 从当前域名开始的绝对链接（以 {{/}} 开头）。
* {{%%content%%}} — 文章内容（标记格式）。
* {{%%rating%%}} — 文章评分。
* {{%%rating_votes%%}} — 文章投票数。
* {{%%current_user_voted%%}} — {{True}}/{{False}}，表示当前用户是否投票。
* {{%%popularity%%}} — 文章人气值：正向投票比例（在点赞系统下）或高于 3.0 的评分比例（在星级系统下）。
* {{%%revisions%%}} — 文章修订次数。
* {{%%index%%}} — 在搜索结果中的序号。
* {{%%total%%}} — 符合条件的文章总数。
* {{%%created_by%%}} — 创建文章的用户名。
* {{%%created_by_linked%%}} — [#user-block 创建者的用户名、头像及个人主页链接]。
* {{%%updated_by%%}} — 最后编辑者用户名。
* {{%%updated_by_linked%%}} — [#user-block 最后编辑者的用户名、头像及个人主页链接]。
* {{%%tags%%}} — 以逗号分隔的标签列表。
* {{%%tags_linked%%}} — 以逗号分隔的标签链接列表，格式为 {{/system:page-tags/tag/标签名}}。
* {{%%created_at%%}} — [#date 创建日期]。
* {{%%updated_at%%}} — [#date 最后修改日期]。

ListPages 模块参数列表：

* {{range}} — 只能使用 {{range="."}} 格式。
用于优化当前页面变量获取。
使用该参数时，其它参数将被忽略。

* {{fullname}} — 限制为指定完整标识符的单一文章。
可使用 {{.}} 表示当前文章。
使用该参数时，其它参数将被忽略。

* {{pagetype}} — 按文章类型筛选。
类型可为 {{hidden}}（标识符以 {{_}} 开头）或 {{normal}}（其它文章）。
默认值为 {{normal}}。

* {{name}} — 按文章自身标识符筛选（不含类别）。
例如 {{name="main"}} 可能匹配 {{wl:main}} 或 {{sandbox:main}}。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{.}} — 当前文章（使用后忽略其它参数）。
 * {{=}} — 与当前文章相同标识符。
 * {{文本%}} 或 {{文本*}} — 指定前缀。
 * 具体标识符 — 精确匹配。

* {{tags}} — 按标签筛选。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{-}} — 无标签文章。
 * {{=}} — 至少包含当前文章标签（允许额外标签）。
 * {{==}} — 标签完全一致。
 * [#iftags iftags 表达式]。

* {{category}} — 按类别筛选。
可接受值：

 * {{*}} — 不限制。
 * {{.}}（默认）— 当前类别。
 * [#ifcategory ifcategory 表达式]。

* {{parent}} — 按父页面筛选。
可接受值：

 * {{-}} — 无父页面。
 * {{=}} — 与当前页面相同父页面。
 * {{-=}} — 与当前页面不同父页面。
 * {{.}} — 父页面为当前页面。
 * 完整标识符 — 指定父页面。

* {{created_by}} — 按作者筛选。
可接受值：

 * {{.}} — 当前用户。
 * 用户名 — 指定作者。

* {{created_at}} — 按创建日期筛选。
格式 {{年-月-日}}（月日可省略）。
支持比较运算：{{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{rating}} — 按评分筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{votes}} — 按投票数筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{popularity}} — 按人气值筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{order}} — 排序。
默认升序。降序需添加 {{desc}}，例如 {{order="created_at desc"}}。
支持排序字段：

 * {{created_at}}
 * {{created_by}}
 * {{updated_at}}
 * {{name}}
 * {{fullname}}
 * {{title}}
 * {{rating}} ##red|(注意：会显著降低查询速度)##
 * {{votes}} ##red|(注意：会显著降低查询速度)##
 * {{popularity}} ##red|(注意：会显著降低查询速度)##
 * {{random}} — 随机排序。

* {{offset}} — 相对于第一篇找到的文章的偏移量，从该位置开始渲染模块；例如 {{offset="1"}} 表示第一篇找到的文章将不会被显示。

* {{limit}} — 模块中最多渲染的文章数量。

* {{perpage}} — 模块单页最多显示的文章数量。
若超过该值，模块中将出现分页列表，可在页面之间切换。
注意：当参数 {{wrapper}} 被设置为负值时，文章数量仍会受到限制，但分页列表 **不会** 出现。

* {{p}} — 当文章数量超过一页时，模块的初始页码。

此外，还可以使用以下参数控制文章信息的输出方式：

* {{prependLine}} — 此参数中的标记将在文章列表渲染之前输出；通常用于表格标题行。
该参数中不支持变量。

* {{appendLine}} — 此参数中的标记将在文章列表渲染完成后输出。
该参数中不支持变量。

* {{separate}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，每篇文章的标记（以及 {{appendLine}} 和 {{prependLine}}）都会作为完全独立的元素处理。
这主要影响 {{@@[[iftags]]@@}} 与 {{@@[[ifcategory]]@@}} 的行为、通过 {{@@[[image]]@@}} 使用这些文章中的图片，以及是否可以将标记拼接为完整代码（例如用于表格）。
因此，若使用 ListPages 渲染表格，必须设置 {{separate="false"}}。

* {{wrapper}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，模块内容会被包裹在带有 {{list-pages-box}} 类名的 {{<div>}} 元素中。
同时启用 ListPages 的动态（AJAX）功能，例如分页切换。
不建议关闭此参数。

* {{reverse}} — [#booleans 布尔值]（默认值为 {{false}}）；启用后，文章列表顺序将被反转。
该操作将在通过 {{order}} 参数完成普通排序之后执行。

对于上述任意属性，都可以通过 {{@URL@@|@@默认值}} 从页面地址中的变量获取值。
该结构会将模块属性设置为与属性同名的 URL 变量值，若未指定则使用 {{默认值}}。例如，若目标文章包含 {{@@[[module ListPages category="@URL@@|@@sandbox"]]@@}}，并通过 {{/category/fragment}} 访问页面，则模块将输出 {{fragment}} 分类下的页面；若未传递变量，则输出 {{sandbox}} 分类下的页面。

++ 模块 CountPages

: 支持参数 : ✅
: 支持内容 : ✅

支持与 [#module-listpages 模块 ListPages] 相同的参数，但不会输出任何文章信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%total%%}}, {{%%count%%}} — 符合条件的文章数量。

++ 模块 ListUsers

: 支持参数 : ✅
: 支持内容 : ✅

该模块可用于获取当前用户的信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%number%%}} — 用户 ID。
* {{%%title%%}}, {{%%name%%}} — 用户名。
* {{%%avatar%%}} — 用户头像链接。

默认情况下，若当前用户未登录，模块将不会显示。

可使用布尔参数 {{always}} 防止该行为。
当其为正值时，模块内容仍会显示，但除 {{%%avatar%%}}（默认头像链接）外，其它变量将不可用。

因此，可以使用以下方式检测用户是否已登录：

[[code]]
[[module ListUsers always="yes"]]
  [[if %%number%%]]
    用户已登录
  [[else]]
    用户未登录
  [[/if]]
[[/module]]
[[/code]]

++ 模块 Redirect

: 支持参数 : ✅
: 支持内容 : ❌

将当前页面重定向至另一页面。支持参数：

* {{destination}} — 目标地址。
不会进行标准过滤，仅禁止以 {{data:}} 或 {{javascript:}} 开头的链接。

* {{noredirect}} — 当为正的 [#booleans 布尔值] 时，阻止模块在当前页面生效。
通常在编辑包含该模块的页面时使用（{{/noredirect/true}}）。

示例：

[[code]]
[[module Redirect destination="/scp-1730"]]
[[/code]]

++ 模块 InterWiki

: 支持参数 : ✅
: 支持内容 : ✅

系统模块。用于在侧边栏显示当前页面的多语言翻译。

通过 API [https://crom.avn.sh/ Crom] 实现，由英文 SCP 社区创建并维护。

对于每个找到的翻译，模块内容会被复制并作为标记处理，并进行 [#autoreplace 自动替换]，可使用以下变量：

* {{%%url%%}} — 翻译页面地址。
* {{%%language%%}} — 翻译语言名称或对应维基名称（若同一语言有多个维基），语言由 {{language}} 参数指定。
* {{%%language_native%%}} — 翻译语言在其自身语言中的名称。
* {{%%language_code%%}} — 翻译语言代码（例如 {{en}}, {{ru}}）。

InterWiki 模块参数：

* {{article}} — 要获取翻译列表的文章名称。

* {{prependLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染前输出。
不支持变量。

* {{appendLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染后输出。
不支持变量。

* {{language}} — {{%%language%%}} 所使用的显示语言。

* {{order}} — 排序方向与变量。可使用 {{url}}, {{language}}, {{language_native}}, {{language_code}}。
可添加 {{desc}} 表示降序（例如 {{order="language_native desc"}}）。

* {{omitlanguage}} — 在获取翻译列表时忽略的语言（通常为当前维基语言）。

* {{empty}} — 若未找到任何翻译时显示的标记。

* {{loading}} — 页面加载完成后立即显示，在获取翻译数据之前显示的标记。

++ 模块 TagCloud 与 PagesByTag

系统模块，用于支持页面 <<[[[system:page-tags|标签云]]]>> 的功能。

++ 模块 SiteChanges

系统模块，用于支持页面 <<[[[system:recent-changes|最近更改]]]>> 的功能。

++ 模块 ForumStart、ForumCategory、ForumThread、ForumNewThread、ForumNewPost

系统模块，用于支持论坛功能。

* ForumStart 模块必须位于系统页面 {{forum:start}}，用于显示分区列表。
* ForumCategory 模块必须位于系统页面 {{forum:category}}，用于显示指定分区的主题列表。
* ForumThread 模块必须位于系统页面 {{forum:thread}}，用于显示指定主题中的帖子。
* ForumNewThread 模块必须位于系统页面 {{forum:new-thread}}，用于在指定分区创建新主题。
* ForumNewPost 不直接用于页面，而通过主题页面的模块 API 使用。

++ 模块 RecentPosts

系统模块，用于支持页面 <<[[[forum:recent-posts|论坛最新帖子]]]>> 的功能。

[[/div]]
', '以下指南是关于PojectWikit网站整体运行机制以及维基标记语言（wiki markup）相关内容的技术文档。

本指南面向具备最低限度 HTML 和 CSS 基础知识的读者；其目标是全面、详尽地描述所有可用功能，并解释为什么某些内容会以目前这种方式运行。

[[toc]]

[[module CSS]]

#page-content h1, #page-content h2, #page-content h3 {
  padding-bottom: 8px;
  border-bottom: 1px solid #aaa;
  margin-top: 32px;
  clear: both;
}

#page-content h1 + h2, #page-content h2 + h3 {
  margin-top: 16px;
}

code, .code, .code pre {
  background: #f7f7f7;
  color: #050;
  font-weight: 500;
  font-family: ''Cascadia Mono'', ''Courier New'', Courier, FreeMono, monospace;
}

code {
  padding: 4px;
  border-radius: 4px;
  white-space: nowrap;
}

.code {
  padding: 8px;
}

.code p, .code pre {
  margin: 0;
}

#page-content dl {
  display: grid;
  grid-template-columns: max-content max-content;
  border: 1px solid #eee;
  border-radius: 8px;
  float: right;
  overflow: hidden;
  margin-left: 32px;
  margin-bottom: 32px;
  background: white;
}

#page-content dl dd, #page-content dl dt {
  padding: 8px;
  border-bottom: 1px solid #eee;
  margin: 0;
}

#page-content dl dd {
  text-align: right;
}

.actual-page-content a[href^="#"] {
  border-bottom: 1px dotted #050;
  color: #050;
  text-decoration: none;
}

.actual-page-content a[href^="#"]:hover {
  border-bottom-style: solid;
}
  @media (max-width: 700px) {
code {
  white-space: wrap;
  word-break: break-all;
}
  }
[[/module]]

[[div class="actual-page-content"]]

+ 引言

标记语言分为四种类型：

* **自动替换** 在文章开始处理之前执行。因此，自动替换允许在文章中添加新的代码，这些代码随后会作为语法被处理。详见 {{[[include]]}} 章节。

* **段落划分** 按既定规则自动进行。

* **自由语法** 没有严格的格式；每个标记元素可能以完全不可预测的方式被解析。这些元素不总是彼此兼容，也不一定与块级元素兼容。

* **块级元素** 具有严格规则；其格式在外观上类似于 HTML 标记或 BBCode。块级元素具有名称、属性、修饰符。那些原则上可以包含其他元素的块级元素，对其所包含元素的类型不作限制（包括其他块级元素）。

+ [[# autoreplace]] 自动替换

++ {{[[include]]}}：从其他文章插入代码

语法：

[[div class="code"]]
@@[[include 文章名称 参数1 = 值1 | 参数2 = 值2]]@@
[[/div]]

为了使该元素正常工作，在起始的 {{[[}} 前面从行首开始不得有任何文本（包括空格）。同样，在结束的 {{]]}} 之后也不能有任何文本。

元素 {{[[include]]}}，以及其中的各个参数，都可以占用多行。

使用该元素时，系统会访问指定的站点文章，获取其源代码，并将该源代码插入到 {{[[include]]}} 所在的位置。

在插入之前，会对指定文章中的所有变量进行自动替换。例如，形如 {{@@{$参数1}@@}} 的变量将被替换为在该元素中指定的对应参数值。

由于插入源代码是在“行级”而非“元素级”进行的，因此被嵌入的文章中可以包含完整或部分源代码。同样，{{[[include]]}} 的参数中也可以包含完整或部分源代码。

示例：

* 文章 {{page1}} 中的代码： _
[[div class="code"]]
@@{$param}@@
[[/div]]

* 使用 {{[[include]]}} 的文章中的代码： _
[[div class="code"]]
@@[[include page1 param=[[div class="code"]] ]]@@
@@text@@
@@[[include page1 param=[[/div]] ]]@@
[[/div]]

* 结果： _
[[div class="code"]]
@@[[div class="code"]]@@
@@text@@
@@[[/div]]@@
[[/div]]

++ {{[[noinclude]]}}：在被插入到其他页面时忽略部分代码

语法：

[[div class="code"]]
@@[[noinclude]]@@
...任意文本...
@@[[/noinclude]]@@
[[/div]]

某些站点组件同时包含可调用代码（组件本身）、使用说明文档以及预览内容。

为了防止这些可视化元素被包含到作者文章中，可以使用 {{[[noinclude]]}} 标签。

无论是起始还是结束的 {{[[noinclude]]}} 标签，都必须单独占据一整行，否则标签不会生效。这样设计是为了降低误触发或错误触发的概率，例如在记录该功能自身文档时。

++ 分类模板

分类模板是形如 [[[component:_template|component:_template]]] 的隐藏页面。对于主分类，页面名称为 [[[_default:_template|_template]]].

如果为某个分类（例如此处的 {{component}}）指定了模板，那么该模板将会为该分类下的所有文章渲染显示，**而不是文章的实际代码**。同时，模板中支持 [#module-listpages ListPages 模块] 中使用的所有变量。例如，可以通过 {{%%content%%}} 获取原始文章代码。

++ [[# path-params]] {{%%path%%}}, {{%%path_expr%%}}, {{%%path_url%%}}：页面参数

站点引擎支持通过形如 {{/参数/值}} 的语法在页面地址中传递参数。

例如，为了在文章 {{page1}} 中访问参数 {{%%param%%}} 和 {{%%param2%%}}，可以通过如下地址访问：

{{@@https://projwikit.unitreaty.org/page1/param/example1/param2/example2@@}}

由于这些变量替换属于自动替换，目标文章可以通过三种方式访问参数：

* {{%%path|param%%}} 直接将变量值插入文章代码；如果未指定该变量，则插入文本 {{%%path|param%%}}。

* {{%%path_expr|param%%}} 以 JSON 字符串格式插入变量值；若未指定，则插入文本 {{"%%path_expr|param%%"}}。这允许在块级元素属性中传递包含特殊字符的复杂值（例如 {{@@[[input type="text" value=%%path_expr|param%%]]@@}} 可确保即便用户使用特殊字符，值也能正确写入）。

* {{%%path_url|param%%}} 插入 URL 编码格式的变量值；若未指定，则插入 {{%25%25path_url%7Cparam%25%25}}。这允许在链接或传递给其他页面的参数中使用这些值（例如 {{@@[[module Redirect to="/other_page/param/%%path_url|param%%"]]@@}}）。

++ 排版符号的自动替换

* {{@<&#96;>@文本@<&#39;>@}} —— 替换为 ‘文本’。

* {{@<&#96;>@@<&#96;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 “文本”。

* {{@<&#44;>@@<&#44;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 „文本”。

[!-- * {{@<&#46;>@@<&#46;>@@<&#46;>@}}, {{@<&#46;>@ @<&#46;>@ @<&#46;>@}} —— 替换为符号 "…". --] [!-- 暂时移除 // jewalky --]

需要注意的是，由于这些符号替换发生在自动替换阶段，因此可能跨多行发生，甚至包括在 [#literals 字面量]、{{@@[[code]]@@}}、{{@@[[module]]@@}} 等内容中；请务必注意。

+ 段落划分

系统中所有可以包含其他元素的元素，分为两大类：

* 行内元素。包括普通文本、所有文本格式元素，以及 {{@@[[span]]@@}} 和其他诸如 {{@@[[image]]@@}}、{{@@[[user]]@@}} 等元素。一般来说，如果该元素默认显示为 {{display: inline}} 或 {{display: inline-block}}，则可视为行内元素。

* 全宽元素。包括标题、分隔线、列表、{{@@[[toc]]@@}}、{{@@[[div]]@@}}、{{@@[[blockquote]]@@}}、{{@@[[footnoteblock]]@@}}、{{@@[[collapsible]]@@}} 等。大致对应 {{display: block}}。

尽管上文提及 CSS 属性，但该属性的实际值不会影响段落生成，因为元素的分类是在其从标记转换为 HTML 的初始阶段完成的。

段落会被创建：

* 在全宽块级元素中，如果未为其指定修饰符 {{_}}（例如 {{@@[[div_]]@@}}）。_
该修饰符并非对所有块级元素都可用，详见各元素说明。

* 在简单引用块（{{>}}）中。

段落不会被创建：

* 在任何行内元素中。

* 在块级表格（{{@@[[table]]@@}}）中。

* 在大多数自由语法元素中（{{>}} 除外）。_
可以通过将所需文本包裹在 {{@@[[div]]@@}} 或 {{@@[[p]]@@}} 中来绕过该限制，例如：_
[[code]]|| [[div]]第一行

第二行[[/div]] || 下一个表格单元格 ||[[/code]]在此示例中，{{@@[[div]]@@}} 内的内容会被包裹为段落，而下一个表格单元格则会被直接作为文本添加。_
该技巧同样适用于块级表格。

要在支持段落的元素中创建或分隔新段落，需要满足以下多个条件：

* 该行必须是元素中的第一行，或者其上方至少有一整行空行，或者其上方存在一个全宽元素。

* 该行必须仅包含行内元素。全宽元素周围不会创建段落。在段落内部插入全宽元素会在该处终止当前段落，并在该全宽元素之后创建新段落。

如果在不创建段落的元素中存在文本（根据上述任一条件），文本中的空行将被视为普通换行（{{<br>}}，而不是 {{<p>}}）。

++ 控制换行

如果你希望空行仅作为空行，而不是创建段落，可以使用两种方法：

* 在该行放置任何视觉上为空的元素（但在段落判定上不视为空）。例如 {{@@[[span]][[/span]]@@}}、{{@<@>@@<@>@@<@>@@<@>@}}、{{@<&#64;&#60;>@@<&#62;&#64;>@}}。

* 在该行末尾添加符号 {{_}}。该符号会被明确解释为换行，并且绝不会转换为段落。

你也可以在代码分成多行时阻止换行（以及段落创建）。这在编写复杂代码时非常有用，可以保持可读性，同时不在视觉上拆分文本。为此，请在行末添加 {{\}}，则下一行会“粘连”到当前行。例如，下列代码在插入文章后将显示为一行 “abc”：

[[div class="code"]]
@@a\@@
@@[[span class="some-class"]]\@@
@@b\@@
@@[[/span]]\@@
@@c@@
[[/div]]

+ 自由语法

++ 文本格式

* {{@@**文本**@@}} —— **粗体** 文本。

* {{@@//文本//@@}} —— //斜体// 文本。

* {{@@{{文本}}@@}} —— 等宽文本。

* {{@@--文本--@@}} —— --删除线-- 文本。

* {{@@^^文本^^@@}} —— ^^上标^^ 文本。

* {{@@,,文本,,@@}} —— ,,下标,, 文本。

* {{@@__文本__@@}} —— __下划线__ 文本。

所有上述文本格式化方式都遵循相同的规则：

* 元素与其内部文本之间不得有空格（例如，{{@@__ 文本 __@@}} 不是正确语法）。

* 元素可以跨多行，但不能跨多个段落。
  正确：
[[code]]//a
b
c//[[/code]]
  错误：
[[code]]//a

b

c//[[/code]]

* 元素内部可以包含任何其他元素，包括块级元素。在使用块级元素的情况下，段落限制将被解除。例如：
[[code]]//[[div]]a

b

c[[/div]]//[[/code]]

同时也支持文本着色：

* {{@@##颜色|文本##@@}} — ##red|有颜色的## 文本。
  颜色可以使用任何 CSS 支持的颜色，例如表达式 {{@@##rgba(255, 127, 0, 0.5)|文本##@@}} 是合法的。
  当使用十六进制值表示颜色（RRGGBB、RRGGBBAA、RGB、RGBA）时，不必额外使用 {{#}} 符号（例如：{{@@##ffe|文本##@@}} 与 {{@@###ffe|文本##@@}} 等效）。

++ [[# links]] 链接

[[ul]]
[[li]]{{@@[地址 文本]@@}} 或 {{[*地址 文本]}} — 普通链接。
通常用于外部链接，或在站内文章之间使用参数进行链接，或用于锚点链接（例如：{{@@[#toc-0 指向第一个标题的链接]@@}}）。
地址不能包含空格，而链接文本可以包含空格。链接文本不得跨多行。
如果链接以星号 {{*}} 开头，则会在新窗口（标签页）中打开。
[[/li]]

[[li]]{{@@[[[文章]]]@@}} 或 {{@@[[[文章|]]]@@}} 或 {{@@[[[文章|链接文本]]]@@}} — 使用完整标识符（地址）创建站内文章链接。
此类链接会显示在文章的反向链接中；如果目标文章不存在，则会以不同颜色显示。

语法变体：

[[ul]]
  [[li]]{{@@[[[category:page1]]]@@}} — 创建指向 {{category:page1}} 的链接，链接文本大致对应文章标识符（符号 {{-}} 会被替换为空格，分类前缀会被移除等）。
  [[/li]]

  [[li]]{{@@[[[category:page1|]]]@@}} — 创建指向 {{category:page1}} 的链接。
  如果该文章存在，则自动使用其标题作为链接文本；否则使用其标识符。
  [[/li]]

  [[li]]{{@@[[[category:page1|文本]]]@@}} — 创建指向 {{category:page1}} 的链接，并使用指定文本作为链接名称。
  链接文本同样不得跨多行。
  [[/li]]
[[/ul]]
[[/li]]

[[li]]自动链接：形如 {{@@http://...@@}}、{{@@ftp://...@@}} 的文本会自动转换为对应链接。
这种自动替换可能会引发问题并破坏其他语法。如需避免，可以使用 [#literals 字面量]。
[[/li]]
[[/ul]]

++ 元素位置控制

* {{@@= 文本@@}} — 段落居中。必须放在段落开头使用，此后整个段落（包括后续行）都会居中对齐。

* {{@@_@@}} — 显式换行。只能用于行尾。
例如，可用于防止两个连续空行被合并为一个段落，如下所示：
[[code]]a
_
_
b[[/code]]

此外，{{@@_@@}} 允许在原本不支持换行的元素中插入换行（例如表格或列表）。
为了正确生效，显式换行符必须与前面的文本或元素之间用空格分隔。

* {{@@~~~@@}}、{{@@~~~<@@}}、{{@@~~~>@@}} — 清除浮动元素。
仅在行首使用时有效。
分别等同于：
{{@@[[div style="clear: both"]][/div]]@@}}
{{@@[[div style="clear: left"]][[/div]]@@}}
{{@@[[div style="clear: right"]][[/div]]@@}}

++ [[# headers]] 标题

{{@@+ 文本@@}}、
{{@@++ 文本@@}}、
{{@@+++ 文本@@}}、
{{@@++++ 文本@@}}、
{{@@+++++ 文本@@}}、
{{@@++++++ 文本@@}} — 分别对应一级至六级标题。

不支持六级以上标题。

该元素只能在新行使用。
内部不支持换行，但可以嵌入块级元素或显式换行，例如：

[[code]]++ 第一行 _
第二行[[/code]]

在标题文本前可以添加符号 {{*}}，例如 {{@@++* 文本@@}}。
此时该标题不会被加入目录。

++ 水平线

{{@@---@@}} — 添加水平分隔线。

该元素只能在新行使用。
前三个 {{-}} 之后的数量没有限制。

++ 引用

以下语法会创建两个嵌套的引用块：

[[code]]
> 第一级
>> 第二级
>> 第二级的第二行
> 回到第一级
[[/code]]

由于 {{@@[[blockquote]]@@}} 具有更好的可读性（和可编辑性），因此不推荐使用此语法。

++ 列表

网站支持三种列表类型：有序列表、无序列表和字典列表。
前两种可以相互嵌套。

示例：无序列表中嵌套有序列表：

[[code]]
* 元素1
* 元素2
 # 元素2.1
 # 元素2.2
* 元素3
[[/code]]

列表的嵌套级别由 {{*}} 或 {{#}} 前的空格或制表符数量决定。

列表项中可以通过 {{_}} 或使用 {{@@[[span]]@@}} 包裹内容来实现多行，例如：

[[code]]
* 元素1 _
下一行
* [[span]]元素2
下一行[[/span]]
[[/code]]

字典列表定义如下：

[[code]]
: 术语1 : 定义
: 术语2 : 定义
: 术语3 : 定义
[[/code]]

在术语或定义中同样可以使用 {{_}} 或 {{@@[[span]]@@}}。

++ 表格

简化（自由）语法的表格如下所示：

[[code]]
||~ 标题 ||~ 标题2 ||
||> 右对齐文本 ||= 居中文本 ||
|||| 横向跨越两列的单元格 ||
[[/code]]

要在单元格中添加多行文本，可以使用 {{_}} 或 {{@@[[span]]@@}}。

该元素无法创建跨越多行（纵向合并）的单元格。在这种情况下，可以使用块级元素 {{@@[[table]]@@}}，它不受此限制。

++ 其他

* {{@@[[# anchor]]@@}} — 创建一个具有指定标识符的元素，从而可以通过链接跳转到该位置（例如：{{@@[#anchor 跳转到锚点]@@}}）。
  该元素在视觉上类似于块级元素，但实际上并不是。
  在元素定义中，{{#}} 后必须保留一个空格。

* {{@@[!-- 注释 --]@@}} — 定义一段在最终页面显示时不会呈现的源代码区域。
  可用于在代码中添加技术性备注。
  [!-- 我就知道你会看到这里。 --]

* [[# literals]]{{@<&#64;&#64;>@文本@<&#64;&#64;>@}} — 阻止 {{@<&#64;&#64;>@}} 包裹的内容被当作标记语法解析。
  始终生成一行文本（字面量）。
  可用于在出于视觉效果使用标记符号时避免歧义（例如，并非链接用途的单个方括号）。
  也可用于破坏自动替换语法（当不希望发生自动替换时），例如：
  {{%%pat@<&#64;&#64;>@h|param%%}} 始终会显示为文本 %%pat@@@@h|param%%，即便指定了参数 {{param}}。
  也可用于像 {{_}} 那样创建空行（但不推荐这样使用）。

* {{@<&#64;&#60;>@&mdash;&copy;@<&#62;&#64;>@}} — 插入指定的 HTML 实体符号（或多个符号）。

* 符号 «、» 和 — 的替换
_
_
由于这些替换不属于自动替换，因此在仅支持纯文本的场景下不会生效（例如块属性或链接名称中）。

 * {{@@ <<@@}} — 左引号：«

 * {{@@ >>@@}} — 右引号：»

 * {{@@ --@@}} — 长破折号：—
   需要注意的是，为了使该元素被解析为破折号而不是删除线语法，其两侧必须有空格。

+ 块级元素

所有块级元素都遵循类似规则构建。

每个块级元素都有名称（例如 {{div}}、{{iftags}}、{{blockquote}} 等）、可选的标识符，以及一个起始标签（例如 {{@@[[div]]@@}}）。
可以包含文本或其他块级元素的块级元素，还必须有与起始标签对应名称的结束标签（例如 {{@@[[/div]]@@}}）。

某些块可以使用修饰符 {{_}}。
该修饰符写在块名称之后，用于阻止在块内自动创建段落（文本会直接放入块中，换行通过 {{<br>}} 实现）。
修饰符只写在起始标签中，因此以下语法是正确的：
[[code]][[div_]]text[[/div]][[/code]]

块的标识符是可选文本，写在块名称之后（但在修饰符 {{_}} 之前），通过 {{:}} 指定。例如：

[[code]]
[[module:lu ListUsers]]
  [[module CSS]]
    body {
      background: url(%%avatar%%);
    }
  [[/module]]
[[/module:lu]]
[[/code]]

块标识符允许在接受文本内容的块（如 {{@@[[code]]@@}}、{{@@[[module]]@@}}、{{@@[[html]]@@}}）内部使用该块的标准结束标签，而不会真正关闭它。
这样可以将多个模块相互嵌套，或在该块内部写出 {{@@[[code]]@@}} 的示例：

[[code:outer]]
[[code:2]]
  [[code:b]]
    示例：使用块 [[cоde]]
  [[/code:b]]
[[/code:2]]
[[/code:outer]]

大多数块可以以某种形式接受属性（要么是 HTML 属性，要么是特定块自定义属性）。
属性写法为 {{参数=值}} 或 {{参数="值"}}。
不同于 HTML，在属性值中使用特殊字符时，不使用 HTML 实体（如 {{&quot;}}），而是通过 {{\}} 转义：
例如 {{@@[[collapsible show="协议 \"忧郁\""]]@@}}。

++ [[# html-attributes]] 标准 HTML 属性

某些元素（例如 {{@@[[a]]@@}}、{{@@[[span]]@@}} 等）是 HTML 的直接接口，
其标记中的属性会直接插入生成的 HTML 页面中。

并非所有 HTML 属性都允许在标记中使用。
允许使用的属性列表如下；更多详情请参阅
https://www.w3schools.com/tags/ref_attributes.asp 的 HTML 文档。
在本网站语境中最常用的属性已用 ##red|红色## 标出。

* {{##red|alt##}}
* {{##red|class##}}
* {{##red|colspan##}}
* {{##red|href##}}
* {{##red|id##}}：该属性会被特殊处理。标识符必须以 {{u-}} 作为前缀。如果未指定此前缀，系统会自动添加。例如，{{id="myid"}} 将会被转换为 {{id="u-myid"}}。
* {{##red|rowspan##}}
* {{##red|style##}}
* {{##red|target##}}
* {{accept}}
* {{align}}
* {{autocapitalize}}
* {{autoplay}}
* {{background}}
* {{bgcolor}}
* {{border}}
* {{buffered}}
* {{checked}}
* {{cite}}
* {{cols}}
* {{contenteditable}}
* {{controls}}
* {{coords}}
* {{datetime}}
* {{decoding}}
* {{default}}
* {{dir}}
* {{dirname}}
* {{disabled}}
* {{download}}
* {{draggable}}
* {{for}}
* {{form}}
* {{headers}}
* {{height}}
* {{hidden}}
* {{high}}
* {{hreflang}}
* {{inputmode}}
* {{ismap}}
* {{itemprop}}
* {{kind}}
* {{label}}
* {{lang}}
* {{list}}
* {{loop}}
* {{low}}
* {{max}}
* {{maxlength}}
* {{min}}
* {{minlength}}
* {{multiple}}
* {{muted}}
* {{name}}
* {{optimum}}
* {{pattern}}
* {{placeholder}}
* {{poster}}
* {{preload}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{role}}
* {{rows}}
* {{scope}}
* {{selected}}
* {{shape}}
* {{size}}
* {{sizes}}
* {{span}}
* {{spellcheck}}
* {{src}}
* {{srclang}}
* {{srcset}}
* {{start}}
* {{step}}
* {{tabindex}}
* {{title}}
* {{translate}}
* {{type}}
* {{usemap}}
* {{value}}
* {{width}}
* {{wrap}}
* {{scrolling}}
* {{frameborder}}

++ [[# booleans]] 布尔属性

文档中标记为可接受布尔值的属性，在实际使用中可以用以下字符串表示：

True：

* {{true}}
* {{t}}
* {{1}}
* {{yes}}

False：

* {{false}}
* {{f}}
* {{0}}
* {{no}}

同样适用于以下内置 HTML 属性：

* {{allowfullscreen}}
* {{allowpaymentrequest}}
* {{async}}
* {{autofocus}}
* {{autoplay}}
* {{checked}}
* {{controls}}
* {{default}}
* {{disabled}}
* {{formnovalidate}}
* {{hidden}}
* {{ismap}}
* {{itemscope}}
* {{loop}}
* {{multiple}}
* {{muted}}
* {{nomodule}}
* {{novalidate}}
* {{open}}
* {{playsinline}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{selected}}
* {{truespeed}}

++ {{@@[[<]]@@}}、{{@@[[>]]@@}}、{{@@[[=]]@@}}、{{@@[[==]]@@}}：对齐

: 类型 : 全宽
: 段落 : ✅
: 支持属性 : ❌

* {{@@[[<]]@@}} — 将内部文本左对齐。

* {{@@[[>]]@@}} — 将内部文本右对齐。

* {{@@[[=]]@@}} — 将内部文本居中对齐。

* {{@@[[==]]@@}} — 将内部文本两端对齐。

++ {{@@[[a]]@@}}：链接

: 类型 : 行内
: 别名 : {{@@[[anchor]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{*}} : ✅
: 支持 {{_}} : ✅

使用修饰符 {{*}}（例如 {{@@[[*a href="https://google.com"]]Google[[/a]]@@}}）等同于使用 {{target="_blank"}}；该链接会在新窗口（标签页）中打开。
若同时使用该修饰符和 {{target}}，其值将会叠加。

通过该元素创建的链接会经过 [#link-handling 标准过滤]。

++ {{@@[[blockquote]]@@}}：引用块

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[quote]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

在功能上等同于使用 {{>}}，但在标记层面更加“干净”和易读。

++ {{@@[[b]]@@}}：加粗文本

: 类型 : 行内
: 别名 : {{@@[[bold]]@@}}、{{@@[[strong]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[char]]@@}}：HTML 字符

: 类型 : 行内
: 别名 : {{@@[[character]]@@}}
: 支持属性 : ❌

在文本中插入一个 HTML 实体字符。
其工作方式与 {{@<&#64;&#60;>@@<&#62;&#64;>@}} 语法几乎相同。

使用示例：{{@@[[char &mdash;]]@@}}

++ {{@@[[code]]@@}}：代码块

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许忽略 {{@@[[code]]@@}} 与 {{@@[[/code]]@@}} 之间的标记规则。
主要用于展示标记示例而不被立即解析。

该元素也可用于代码高亮。

通过该元素添加到页面的代码，可以通过如下格式的单独文件访问：

{{@@https@@://files.projwikit.unitreaty.org/local@@--@@code/<页面名称>/<页面中的代码块编号，从 1 开始>}}

对于 HTML、JavaScript、XML、CSS 语言，将设置对应的 MIME 类型，
从而可以在 {{<script src>}}、{{<link rel>}} 等需要类型匹配的场景中使用。

仅支持一个属性：

* {{type}} — 需要高亮的编程语言名称。

[[collapsible show="[+] 支持语言完整列表" hide="[-] 支持语言完整列表"]]

||~ 语言名称                ||~ 缩写                ||
|| 1C                      || 1c                     ||
|| ABNF                    || abnf                   ||
|| Access logs             || accesslog              ||
|| Ada                     || ada                    ||
|| Arduino (C++ w/Arduino libs) || arduino, ino           ||
|| ARM assembler           || armasm, arm            ||
|| AVR assembler           || avrasm                 ||
|| ActionScript            || actionscript, as       ||
|| AngelScript             || angelscript, asc       ||
|| Apache                  || apache, apacheconf     ||
|| AppleScript             || applescript, osascript ||
|| Arcade                  || arcade                 ||
|| AsciiDoc                || asciidoc, adoc         ||
|| AspectJ                 || aspectj                ||
|| AutoHotkey              || autohotkey             ||
|| AutoIt                  || autoit                 ||
|| Awk                     || awk, mawk, nawk, gawk  ||
|| Bash                    || bash, sh, zsh          ||
|| Basic                   || basic                  ||
|| BNF                     || bnf                    ||
|| Brainfuck               || brainfuck, bf          ||
|| C#                      || csharp, cs             ||
|| C                       || c, h                   ||
|| C++                     || cpp, hpp, cc, hh, c++, h++, cxx, hxx ||
|| C/AL                    || cal                    ||
|| Cache Object Script    || cos, cls               ||
|| CMake                   || cmake, cmake.in        ||
|| Coq                     || coq                    ||
|| CSP                     || csp                    ||
|| CSS                     || css                    ||
|| Cap’n Proto             || capnproto, capnp       ||
|| Clojure                 || clojure, clj           ||
|| CoffeeScript            || coffeescript, coffee, cson, iced ||
|| Crmsh                   || crmsh, crm, pcmk       ||
|| Crystal                 || crystal, cr            ||
|| D                       || d                      ||
|| Dart                    || dart                   ||
|| Delphi                  || dpr, dfm, pas, pascal  ||
|| Diff                    || diff, patch            ||
|| Django                  || django, jinja          ||
|| DNS Zone file           || dns, zone, bind        ||
|| Dockerfile              || dockerfile, docker     ||
|| DOS                     || dos, bat, cmd          ||
|| dsconfig                || dsconfig               ||
|| DTS (Device Tree)       || dts                    ||
|| Dust                    || dust, dst              ||
|| EBNF                    || ebnf                   ||
|| Elixir                  || elixir                 ||
|| Elm                     || elm                    ||
|| Erlang                  || erlang, erl            ||
|| Excel                   || excel, xls, xlsx       ||
|| F#                      || fsharp, fs             ||
|| FIX                     || fix                    ||
|| Fortran                 || fortran, f90, f95      ||
|| G-Code                  || gcode, nc              ||
|| Gams                    || gams, gms              ||
|| GAUSS                   || gauss, gss             ||
|| Gherkin                 || gherkin                ||
|| Go                      || go, golang             ||
|| Golo                    || golo, gololang         ||
|| Gradle                  || gradle                 ||
|| GraphQL                 || graphql                ||
|| Groovy                  || groovy                 ||
|| HTML, XML               || xml, html, xhtml, rss, atom, xjb, xsd, xsl, plist, svg ||
|| HTTP                    || http, https            ||
|| Haml                    || haml                   ||
|| Handlebars              || handlebars, hbs, html.hbs, html.handlebars        ||
|| Haskell                 || haskell, hs            ||
|| Haxe                    || haxe, hx               ||
|| Hy                      || hy, hylang             ||
|| Ini, TOML               || ini, toml              ||
|| Inform7                 || inform7, i7            ||
|| IRPF90                  || irpf90                 ||
|| JSON                    || json                   ||
|| Java                    || java, jsp              ||
|| JavaScript              || javascript, js, jsx    ||
|| Julia                   || julia, julia-repl      ||
|| Kotlin                  || kotlin, kt             ||
|| LaTeX                   || tex                    ||
|| Leaf                    || leaf                   ||
|| Lasso                   || lasso, ls, lassoscript ||
|| Less                    || less                   ||
|| LDIF                    || ldif                   ||
|| Lisp                    || lisp                   ||
|| LiveCode Server         || livecodeserver         ||
|| LiveScript              || livescript, ls         ||
|| Lua                     || lua                    ||
|| Makefile                || makefile, mk, mak, make ||
|| Markdown                || markdown, md, mkdown, mkd ||
|| Mathematica             || mathematica, mma, wl   ||
|| Matlab                  || matlab                 ||
|| Maxima                  || maxima                 ||
|| Maya Embedded Language  || mel                    ||
|| Mercury                 || mercury                ||
|| Mizar                   || mizar                  ||
|| Mojolicious             || mojolicious            ||
|| Monkey                  || monkey                 ||
|| Moonscript              || moonscript, moon       ||
|| N1QL                    || n1ql                   ||
|| NSIS                    || nsis                   ||
|| Nginx                   || nginx, nginxconf       ||
|| Nim                     || nim, nimrod            ||
|| Nix                     || nix                    ||
|| OCaml                   || ocaml, ml              ||
|| Objective C             || objectivec, mm, objc, obj-c, obj-c++, objective-c++ ||
|| OpenGL Shading Language || glsl                   ||
|| OpenSCAD                || openscad, scad         ||
|| Oracle Rules Language   || ruleslanguage          ||
|| Oxygene                 || oxygene                ||
|| PF                      || pf, pf.conf            ||
|| PHP                     || php                    ||
|| Parser3                 || parser3                ||
|| Perl                    || perl, pl, pm           ||
|| Plaintext               || plaintext, txt, text   ||
|| Pony                    || pony                   ||
|| PostgreSQL & PL/pgSQL   || pgsql, postgres, postgresql ||
|| PowerShell              || powershell, ps, ps1    ||
|| Processing              || processing             ||
|| Prolog                  || prolog                 ||
|| Properties              || properties             ||
|| Protocol Buffers        || protobuf               ||
|| Puppet                  || puppet, pp             ||
|| Python                  || python, py, gyp        ||
|| Python profiler results || profile                ||
|| Python REPL             || python-repl, pycon     ||
|| Q                       || k, kdb                 ||
|| QML                     || qml                    ||
|| R                       || r                      ||
|| ReasonML                || reasonml, re           ||
|| RenderMan RIB           || rib                    ||
|| RenderMan RSL           || rsl                    ||
|| Roboconf                || graph, instances       ||
|| Ruby                    || ruby, rb, gemspec, podspec, thor, irb ||
|| Rust                    || rust, rs               ||
|| SAS                     || SAS, sas               ||
|| SCSS                    || scss                   ||
|| SQL                     || sql                    ||
|| STEP Part 21            || p21, step, stp         ||
|| Scala                   || scala                  ||
|| Scheme                  || scheme                 ||
|| Scilab                  || scilab, sci            ||
|| Shell                   || shell, console         ||
|| Smali                   || smali                  ||
|| Smalltalk               || smalltalk, st          ||
|| SML                     || sml, ml                ||
|| Stan                    || stan, stanfuncs        ||
|| Stata                   || stata                  ||
|| Stylus                  || stylus, styl           ||
|| SubUnit                 || subunit                ||
|| Swift                   || swift                  ||
|| Tcl                     || tcl, tk                ||
|| Test Anything Protocol  || tap                    ||
|| Thrift                  || thrift                 ||
|| TP                      || tp                     ||
|| Twig                    || twig, craftcms         ||
|| TypeScript              || typescript, ts         ||
|| VB.Net                  || vbnet, vb              ||
|| VBScript                || vbscript, vbs          ||
|| VHDL                    || vhdl                   ||
|| Vala                    || vala                   ||
|| Verilog                 || verilog, v             ||
|| Vim Script              || vim                    ||
|| X++                     || axapta, x++            ||
|| x86 Assembly            || x86asm                 ||
|| XL                      || xl, tao                ||
|| XQuery                  || xquery, xpath, xq      ||
|| YAML                    || yml, yaml              ||
|| Zephir                  || zephir, zep            ||

[[/collapsible]]

++ {{@@[[collapsible]]@@}}：可折叠区块

: 类型 : 全宽
: 段落 : ✅
: 支持 HTML 属性 : ❌

创建一个可通过按钮展开或折叠的区块。

除 HTML 属性外，还支持以下属性：

* {{show}} — 区块关闭时按钮显示的文本。

* {{hide}} — 区块打开时按钮显示的文本。

* {{align}} — 按钮文本对齐方式。可选值：
  {{left}}、{{right}}、{{center}}、{{justify}}。

* {{folded}} — 指示区块是否默认展开。[#booleans 布尔值]。

* {{hideLocation}} — 指示展开后关闭按钮的显示位置。
  可选值：{{top}}、{{bottom}}、{{both}}、{{neither}}、{{none}}。
  后两个选项等价。

++ [[# date]] {{@@[[date]]@@}}：日期

: 类型 : 行内
: 支持 HTML 属性 : ❌

插入日期，并根据查看者计算机的时区自动调整显示。

使用特殊属性语法；日期直接写在块名后，而非单独属性：

[[code]]
[[date 2024-02-18T00:00:00Z format="%H:%M:%S %d.%m.%Y"]]
[[/code]]

例如，该日期以 UTC 指定，
但在 UTC+0200 时区的计算机上将显示为：

"02:00:00 18.02.2024"

++ {{@@[[div]]@@}}：全宽通用容器

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

++ [[# footnotes]] {{@@[[footnote]]@@}}：脚注

: 类型 : 行内
: 支持属性 : ❌

在文本中添加编号脚注。
其内容默认显示在页面底部，或显示在 {{@@[[footnoteblock]]@@}} 所在位置（若手动指定）。

++ {{@@[[footnoteblock]]@@}}：脚注块

: 类型 : 全宽
: 支持 HTML 属性 : ❌

显示文章中所有脚注的列表。

支持以下属性：

* {{title}} — 显示在脚注列表上方的标题文本。

* {{hide}} — [#booleans 布尔值]。
  若指定，该脚注块将不显示。
  可用于移除页面自动添加的默认脚注块，使脚注仅以悬浮提示形式显示。

++ {{@@[[form]]@@}}：表单

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

允许填写表单并通过 [#path-params URL 参数] 提交到站内指定页面。

支持所有常规 HTML 表单属性，
但 {{target}} 属性具有特殊含义：它应填写完整的文章标识符，而不是 URL。

使用示例：

[[code]]
[[form target="search"]]
  [[input type="text" name="s"]]
  [[input type="submit"]]
[[/form]]
[[/code]]

点击“提交”后，将跳转到文章 {{search}}，
例如：

{{@@https://projwikit.unitreaty.org/search/s/你的_字符串@@}}

随后可在目标文章中使用 {{@@%%path%%@@}} 功能读取该参数。

++ {{@@[[html]]@@}}: HTML 代码

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许插入任意 HTML 代码，系统会自动将其包裹在 {{@@[[iframe]]@@}} 中。以这种方式创建的框架会自动根据其内容大小进行自适应。

在当前版本的网站中，该元素通过 {{<iframe srcdoc="...">}} 实现。通过这种方式创建的框架没有域名，因此在框架内部的 JS 代码中无法使用 {{window.localStorage}} 和 {{document.cookie}}。

支持以下属性：

* {{external}} — [#booleans 布尔值]（默认 — {{false}}）；启用该参数后，区块将通过媒体域名（https://files.projwikit.unitreaty.org）进行渲染，其方式与 Wikidot 平台相同。此类区块通常加载时间会稍慢几秒，但支持使用 {{window.localStorage}} 和 {{document.cookie}}。
**注意：** 当前页面版本系统在启用 {{external}} 的 HTML 区块中无法正常工作，因此始终会显示你代码的最新版本（包括在预览和查看旧版本页面时）。此外，此类区块不会读取 [#path-params 页面参数]。

++ {{@@[[iframe]]@@}}: 通过链接嵌入外部页面

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

语法：

[[code]]
[[iframe 链接 属性1="值" 属性2="值"]]
[[/code]]

支持所有允许的 HTML 属性。

通过该元素插入的链接会经过 [#link-handling 标准过滤]。

++ [[# if]] {{@@[[#if]]@@}}, {{@@[[if]]@@}}: 根据值是否存在显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

行内示例：

[[code]]
[[#if {$param} | param exists | param does not exist]]
[[/code]]

全宽示例：

[[code]]
[[if {$param}]]
  param exists
[[else]]
  param does not exist
[[/if]]
[[/code]]

只要是非空字符串都被视为“存在”，//除了// {{@@{$变量}@@}} 或 {{@@%%变量%%@@}} 这种格式的字符串。由于未在 {{@@[[include]]@@}} 或模块中传递的参数会保留为文本值，因此可以借此判断参数是否被传入。

++ [[# ifexpr]] {{@@[[#ifexpr]]@@}}, {{@@[[ifexpr]]@@}}: 根据条件表达式显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

块级示例：

[[code]]
[[module CountPages fullname="main"]]
  [[#ifexpr %%count%% > 0 | yes | no]]
[[/module]]
[[/code]]

全宽示例：

[[code]]
[[module CountPages fullname="main"]]
  [[ifexpr %%count%% > 0]]
    main page exists
  [[else]]
    main page does not exist
  [[/ifexpr]]
[[/module]]
[[/code]]

若表达式语法错误（例如变量不存在或参数传递错误），表达式视为 false。

否则，除 {{False}}、{{0}}、{{0.0}} 或 {{""}} 以外的任何值都视为 true。

支持的运算符：{{*}}, {{/}}, {{+}}, {{-}}, {{@@ <<@@}}, {{@@>>@@}}, 以及比较运算 {{==}}, {{!=}}, {{<}}, {{>}}, {{<=}}, {{>=}}。

除直接比较外，大多数运算仅支持数字。字符串必须以 JSON 格式表示（例如 {{"值"}}）。

还支持以下函数：

* {{min(x, y, ...)}} — 返回最小值。
* {{max(x, y, ...)}} — 返回最大值。
* {{abs(x)}} — 返回绝对值。
* {{round(x)}} — 四舍五入为整数。
* {{lower(str)}} — 转为小写。
* {{upper(str)}} — 转为大写。

++ [[# ifcategory]] {{@@[[ifcategory]]@@}}: 根据文章分类显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

示例：

[[code]]
[[ifcategory +theme]]
该文本仅在 theme 分类页面中显示。
[[/ifcategory]]
[[ifcategory -theme]]
该文本在除 theme 外的所有页面显示。
[[/ifcategory]]
[[/code]]

该元素主要用于与 {{[[include]]}} 配合使用，使插入的代码根据所在页面的分类产生不同效果。

若存在多个分类，应以空格分隔。

++ [[# iftags]] {{@@[[iftags]]@@}}: 根据文章标签显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

标签条件写在区块名称后，以空格分隔，格式如下：

* {{+标签}} — 页面必须包含该标签。
* {{-标签}} — 页面不得包含该标签。
* {{标签}} — 页面至少包含列出的其中一个标签。

示例：

[[code]]
[[iftags 对象 故事 +ru]]
该文章属于俄罗斯分部的对象或故事。
[[/iftags]]
[[/code]]

++ {{@@[[image]]@@}}: 图片

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅

插入当前页面附件图片：

[[code]]
[[image img.png alt="my img"]]
[[/code]]

插入其他页面附件图片：

[[code]]
[[image main/ico_arthub.svg alt="image from another page"]]
[[/code]]

也可以插入互联网图片（不推荐，因为可能发生链接失效——上传至站点的文件会随文章长期保留，而网络图片可能随时失效）。

支持对齐修饰符 {{f<}}, {{f>}}, {{<}}, {{>}}, {{=}}。后三种会使图片成为全宽元素。

* {{@@[[f<image]]@@}} — 左浮动图片，文字会环绕。
* {{@@[[f>image]]@@}} — 右浮动图片。
* {{@@[[<image]]@@}} — 左对齐全宽图片。
* {{@@[[>image]]@@}} — 右对齐全宽图片。
* {{@@[[=image]]@@}} — 居中全宽图片。

支持 {{link}} 属性，可将图片自动包裹为链接。该链接会经过 [#link-handling 标准过滤]，效果等同于 {{@@[[a href="..."]][[image ...]][[/a]]@@}}。

++ {{@@[[input]]@@}}: 输入框

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅

可单独使用，也可与表单搭配使用。

++ {{@@[[i]]@@}}: 斜体文本

: 类型 : 行内
: 别名 : {{@@[[italics]]@@}}, {{@@[[em]]@@}}, {{@@[[emphasis]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[lines]]@@}}: 插入空行

: 类型 : 行内
: 支持属性 : ❌

用于插入多行空白。

示例：

[[code]]
[[lines 8]]
[[/code]]

++ {{@@[[ul]]@@}}, {{@@[[ol]]@@}}, {{@@[[li]]@@}}: 列表

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

* {{@@[[ul]]@@}} — 无序列表（全宽）。
* {{@@[[ol]]@@}} — 有序列表（全宽）。
* {{@@[[li]]@@}} — 列表项（行内）。

示例：

[[code]]
[[ul]]
  [[li]]第一项[[/li]]
  [[li]]第二项[[/li]]
  [[li]]
    [[ol]]
      [[li]]第一条编号[[/li]]
      [[li]]第二条编号[[/li]]
    [[/ol]]
  [[/li]]
  [[li]]第三项[[/li]]
[[/ul]]
[[/code]]

所有列表元素均支持标准 HTML 属性。

++ {{@@[[mark]]@@}}: 高亮文本

: 类型 : 行内
: 别名 : {{@@[[highlight]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# module-block]] {{@@[[module]]@@}}: 插入站点模块

: 类型 : 全宽
: 别名 : {{@@[[module654]]@@}}
: 支持 HTML 属性 : ❌

通用语法：

[[code]]
[[module 模块名称 属性1="值" 属性2="值"]]
模块内容（仅适用于支持内容的模块）
[[/module]]
[[/code]]

模块列表及其属性见“模块”章节。

对于不支持内容的模块（如 {{@@[[module Rate]]@@}}），无需写关闭标签。

++ {{@@[[table]]@@}}, {{@@[[row]]@@}}, {{@@[[hcell]]@@}}, {{@@[[cell]]@@}}: 表格

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

类似于标准 HTML 表格。

* {{@@[[table]]@@}} — {{<table>}}
* {{@@[[row]]@@}} — {{<tr>}}
* {{@@[[hcell]]@@}} — {{<th>}}
* {{@@[[cell]]@@}} — {{<td>}}

不支持 {{<thead>}}、{{<tbody>}}、{{<tfoot>}}。

示例：

[[code]]
[[table class="wiki-content-table"]]
  [[row]]
    [[hcell]]标题 1[[/hcell]]
    [[hcell]]标题 2[[/hcell]]
    [[cell rowspan="2"]]跨行单元格[[/cell]]
  [[/row]]
  [[row]]
    [[cell colspan="2"]]跨列单元格[[/cell]]
  [[/row]]
[[/table]]
[[/code]]

++ {{@@[[tabview]]@@}}, {{@@[[tab]]@@}}: 标签页

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[tabs]]@@}}（对应 {{@@[[tabview]]@@}}）
: 支持 HTML 属性 : ❌

用于在页面中创建可切换的标签页。

示例：

[[code]]
[[tabview]]
  [[tab 标签 1]]
    标签 1 内容
  [[/tab]]
  [[tab 标签 2]]
    标签 2 内容
  [[/tab]]
[[/tabview]]
[[/code]]

标签名称也可使用 {{@@[[tab title="标签名称"]]@@}} 格式。

++ {{@@[[tt]]@@}}: 等宽文本

: 类型 : 行内
: 别名 : {{@@[[mono]]@@}}, {{@@[[monospace]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[p]]@@}}：显式段落

: 类型 : 全宽
: 替代名称 : {{@@[[paragraph]]@@}}
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

允许为当前段落指定 HTML 属性，就像为 {{<p>}} 指定属性一样。

示例：

[[code]]
[[p style="color: red"]]
红色段落。
[[/p]]
[[/code]]

++ {{@@[[ruby]]@@}}, {{@@[[rt]]@@}}：汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[rubytext]]@@}}（用于 {{@@[[rt]]@@}}）
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[ruby]]マレニア[[rt]]Malenia[[/rt]][[/ruby]]
[[/code]]

++ {{@@[[rb]]@@}}：简化汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[ruby2]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[rb マレニア | Malenia]]
[[/code]]

++ {{@@[[size]]@@}}：字体大小

: 类型 : 行内
: 支持属性 : ❌

允许修改字体大小。尺寸可以使用任何 CSS 允许的数值。

~~~

使用示例：

[[code]]
这段文字非常[[size 200%]]大[[/size]]，同时又[[size 6px]]小[[/size]]。
[[/code]]

++ {{@@[[span]]@@}}：行内通用容器

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

可以包含任何其他元素，也可用于为文章中的文本添加样式。

++ {{@@[[s]]@@}}：删除线文本

: 类型 : 行内
: 替代名称 : {{@@[[strikethrough]]@@}}, {{@@[[del]]@@}}, {{@@[[deletion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sup]]@@}}：上标文本

: 类型 : 行内
: 替代名称 : {{@@[[super]]@@}}, {{@@[[superscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sub]]@@}}：下标文本

: 类型 : 行内
: 替代名称 : {{@@[[subscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[toc]]@@}}：自动目录

: 类型 : 全宽
: 支持属性 : ❌

在页面中添加一个可展开的标题列表块。

可使用以下前缀改变元素位置：

* {{@@[[f<toc]]@@}} — 左侧浮动块。该块会被周围文本和块级元素环绕。

* {{@@[[f>toc]]@@}} — 右侧浮动块。

++ {{@@[[u]]@@}}：下划线文本

: 类型 : 行内
: 替代名称 : {{@@[[underline]]@@}}, {{@@[[ins]]@@}}, {{@@[[insertion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# user-block]] {{@@[[user]]@@}}：用户链接

: 类型 : 行内
: 支持属性 : ❌
: 支持 {{*}} : ✅

可以以两种形式使用：

* {{@@[[user 用户名]]@@}} — 输出普通用户链接。

* {{@@[[*user 用户名]]@@}} — 输出用户链接及其头像。

+ [[# link-handling]] 补充：链接处理

出于安全原因，某些链接会被网站屏蔽。

除非某个元素或模块的文档中另有说明，否则所有链接（例如 {{@@[]@@}}、{{@@[[[|]]]@@}}、{{@@[[a href="..."]]@@}}、{{@@[[image ... link="..."]]@@}} 等）都会按照以下规则进行检查。

明确 **禁止** 的绝对链接协议：

* {{data:}}
* {{javascript:}}（除 {{"javascript:;"}} 之外，该形式表示“空链接”）

明确允许的绝对链接协议：

* {{blob:}}
* {{@@chrome-extension://@@}}
* {{@@chrome://@@}}
* {{@@content://@@}}
* {{data:}}
* {{dns:}}
* {{feed:}}
* {{@@file://@@}}
* {{@@ftp://@@}}
* {{@@git://@@}}
* {{@@gopher://@@}}
* {{@@http://@@}}
* {{@@https://@@}}
* {{@@irc6://@@}}
* {{@@irc://@@}}
* {{@@ircs://@@}}
* {{mailto:}}
* {{@@resource://@@}}
* {{@@rtmp://@@}}
* {{@@sftp://@@}}

对于块级元素，下列以指定字符开头的链接同样被视为合法链接：

* {{A-Z}}, {{a-z}}, {{0-9}}, {{.}}（普通相对链接）。
* {{/}}, {{@@//@@}}（基于当前域名和协议的绝对链接）。
* {{#}}（锚点跳转）。
* {{?}}（跳转到当前页面并附带 GET 参数）。
* {{$}}, {{&}}, {{+}}, {{,}}, {{:}}, {{;}}, {{=}}, {{@}}, {{%}}, {{-}}, {{~}}（其他允许用于相对链接的特殊字符）。

对于自由语法元素，会采用更严格的校验规则，并允许更少的特殊字符，以便将链接与语法本身区分开来：

* {{#}} + ({{A-Z}}, {{a-z}}, {{0-9}}, {{_}}, {{-}}, {{%}})（锚点跳转）。
* {{@@//地址@@}}（使用当前协议的绝对链接）。
* {{/地址}}（使用当前域名的绝对链接）。
* 任何不以 {{/}} 开头，但包含它的文本。

+ 补充：模块

模块提供了网站中不属于普通文章的全部功能。这包括交互元素（评分、论坛、标签云）以及扩展功能（向文章添加 CSS 样式、获取其他文章或当前用户的信息等）。

如果模块的运行导致网站功能异常，可以通过添加参数 {{/nomodule/true}} 访问页面（例如：{{@@https://projwikit.unitreaty.org/main/nomodule/true@@}}）。
该参数会完全禁用此页面上的 **所有** 模块。

关于如何在文章中插入模块的详细说明，请参见 [#module-block 章节] {{@@[[module]]@@}}。

此外，需要注意的是，对于支持内部标记的模块（例如 {{ListUsers}}、{{ListPages}} 和 {{CountPages}}），模块内容在技术上并不属于文章内容的一部分。例如，在模块内部定义的任何 [#headers 标题] 或 [#footnotes 脚注]，都只会在该模块范围内显示。

++ 模块 Rate

: 支持参数 : ❌
: 支持内容 : ❌

为文章添加评分组件，效果类似于页面底部“评分”按钮下方的评分模块。

从发布规则角度来看，该模块是必需的，但从技术角度来看并非强制。如果文章中未包含该模块，仍然可以通过“评分”按钮进行投票。

该模块不接受任何参数。

++ 模块 CSS

: 支持参数 : ❌
: 支持内容 : ✅

该模块的内容会原样作为 CSS 样式应用到当前页面。例如：

[[code]]
[[module CSS]]
body {
  background: red;
}
[[/module]]
[[/code]]

可以通过如下地址获取页面上所有 CSS 模块合并后的结果文件：
{{@@https@@://files.projwikit.unitreaty.org/local@@--@@theme/<页面名称>/style.css}}

该方法会处理 {{@@[[if]]@@}}、{{@@[[ifexpr]]@@}} 和 {{@@[[noinclude]]@@}}。
可以通过参数 {{?includeParams=<JSON 格式参数>}} 传递参数。

++ [[# module-listpages]] 模块 ListPages

: 支持参数 : ✅
: 支持内容 : ✅

用于根据指定参数获取文章列表。

对于每一篇找到的文章，模块内容都会被复制并作为标记进行处理，并进行 [#autoreplace 自动替换]，替换以下变量：

* {{%%name%%}} — 文章的自身标识符（例如 {{main}}）。
* {{%%category%%}} — 文章类别（例如 {{sandbox}}）。
* {{%%fullname%%}} — 包含类别的完整标识符（例如 {{sandbox:main}}）。
* {{%%title%%}} — 文章标题。
* {{%%title_linked%%}} — 包含标题的 [#links 内部链接]。
* {{%%link%%}} — 从当前域名开始的绝对链接（以 {{/}} 开头）。
* {{%%content%%}} — 文章内容（标记格式）。
* {{%%rating%%}} — 文章评分。
* {{%%rating_votes%%}} — 文章投票数。
* {{%%current_user_voted%%}} — {{True}}/{{False}}，表示当前用户是否投票。
* {{%%popularity%%}} — 文章人气值：正向投票比例（在点赞系统下）或高于 3.0 的评分比例（在星级系统下）。
* {{%%revisions%%}} — 文章修订次数。
* {{%%index%%}} — 在搜索结果中的序号。
* {{%%total%%}} — 符合条件的文章总数。
* {{%%created_by%%}} — 创建文章的用户名。
* {{%%created_by_linked%%}} — [#user-block 创建者的用户名、头像及个人主页链接]。
* {{%%updated_by%%}} — 最后编辑者用户名。
* {{%%updated_by_linked%%}} — [#user-block 最后编辑者的用户名、头像及个人主页链接]。
* {{%%tags%%}} — 以逗号分隔的标签列表。
* {{%%tags_linked%%}} — 以逗号分隔的标签链接列表，格式为 {{/system:page-tags/tag/标签名}}。
* {{%%created_at%%}} — [#date 创建日期]。
* {{%%updated_at%%}} — [#date 最后修改日期]。

ListPages 模块参数列表：

* {{range}} — 只能使用 {{range="."}} 格式。
用于优化当前页面变量获取。
使用该参数时，其它参数将被忽略。

* {{fullname}} — 限制为指定完整标识符的单一文章。
可使用 {{.}} 表示当前文章。
使用该参数时，其它参数将被忽略。

* {{pagetype}} — 按文章类型筛选。
类型可为 {{hidden}}（标识符以 {{_}} 开头）或 {{normal}}（其它文章）。
默认值为 {{normal}}。

* {{name}} — 按文章自身标识符筛选（不含类别）。
例如 {{name="main"}} 可能匹配 {{wl:main}} 或 {{sandbox:main}}。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{.}} — 当前文章（使用后忽略其它参数）。
 * {{=}} — 与当前文章相同标识符。
 * {{文本%}} 或 {{文本*}} — 指定前缀。
 * 具体标识符 — 精确匹配。

* {{tags}} — 按标签筛选。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{-}} — 无标签文章。
 * {{=}} — 至少包含当前文章标签（允许额外标签）。
 * {{==}} — 标签完全一致。
 * [#iftags iftags 表达式]。

* {{category}} — 按类别筛选。
可接受值：

 * {{*}} — 不限制。
 * {{.}}（默认）— 当前类别。
 * [#ifcategory ifcategory 表达式]。

* {{parent}} — 按父页面筛选。
可接受值：

 * {{-}} — 无父页面。
 * {{=}} — 与当前页面相同父页面。
 * {{-=}} — 与当前页面不同父页面。
 * {{.}} — 父页面为当前页面。
 * 完整标识符 — 指定父页面。

* {{created_by}} — 按作者筛选。
可接受值：

 * {{.}} — 当前用户。
 * 用户名 — 指定作者。

* {{created_at}} — 按创建日期筛选。
格式 {{年-月-日}}（月日可省略）。
支持比较运算：{{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{rating}} — 按评分筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{votes}} — 按投票数筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{popularity}} — 按人气值筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{order}} — 排序。
默认升序。降序需添加 {{desc}}，例如 {{order="created_at desc"}}。
支持排序字段：

 * {{created_at}}
 * {{created_by}}
 * {{updated_at}}
 * {{name}}
 * {{fullname}}
 * {{title}}
 * {{rating}} ##red|(注意：会显著降低查询速度)##
 * {{votes}} ##red|(注意：会显著降低查询速度)##
 * {{popularity}} ##red|(注意：会显著降低查询速度)##
 * {{random}} — 随机排序。

* {{offset}} — 相对于第一篇找到的文章的偏移量，从该位置开始渲染模块；例如 {{offset="1"}} 表示第一篇找到的文章将不会被显示。

* {{limit}} — 模块中最多渲染的文章数量。

* {{perpage}} — 模块单页最多显示的文章数量。
若超过该值，模块中将出现分页列表，可在页面之间切换。
注意：当参数 {{wrapper}} 被设置为负值时，文章数量仍会受到限制，但分页列表 **不会** 出现。

* {{p}} — 当文章数量超过一页时，模块的初始页码。

此外，还可以使用以下参数控制文章信息的输出方式：

* {{prependLine}} — 此参数中的标记将在文章列表渲染之前输出；通常用于表格标题行。
该参数中不支持变量。

* {{appendLine}} — 此参数中的标记将在文章列表渲染完成后输出。
该参数中不支持变量。

* {{separate}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，每篇文章的标记（以及 {{appendLine}} 和 {{prependLine}}）都会作为完全独立的元素处理。
这主要影响 {{@@[[iftags]]@@}} 与 {{@@[[ifcategory]]@@}} 的行为、通过 {{@@[[image]]@@}} 使用这些文章中的图片，以及是否可以将标记拼接为完整代码（例如用于表格）。
因此，若使用 ListPages 渲染表格，必须设置 {{separate="false"}}。

* {{wrapper}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，模块内容会被包裹在带有 {{list-pages-box}} 类名的 {{<div>}} 元素中。
同时启用 ListPages 的动态（AJAX）功能，例如分页切换。
不建议关闭此参数。

* {{reverse}} — [#booleans 布尔值]（默认值为 {{false}}）；启用后，文章列表顺序将被反转。
该操作将在通过 {{order}} 参数完成普通排序之后执行。

对于上述任意属性，都可以通过 {{@URL@@|@@默认值}} 从页面地址中的变量获取值。
该结构会将模块属性设置为与属性同名的 URL 变量值，若未指定则使用 {{默认值}}。例如，若目标文章包含 {{@@[[module ListPages category="@URL@@|@@sandbox"]]@@}}，并通过 {{/category/fragment}} 访问页面，则模块将输出 {{fragment}} 分类下的页面；若未传递变量，则输出 {{sandbox}} 分类下的页面。

++ 模块 CountPages

: 支持参数 : ✅
: 支持内容 : ✅

支持与 [#module-listpages 模块 ListPages] 相同的参数，但不会输出任何文章信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%total%%}}, {{%%count%%}} — 符合条件的文章数量。

++ 模块 ListUsers

: 支持参数 : ✅
: 支持内容 : ✅

该模块可用于获取当前用户的信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%number%%}} — 用户 ID。
* {{%%title%%}}, {{%%name%%}} — 用户名。
* {{%%avatar%%}} — 用户头像链接。

默认情况下，若当前用户未登录，模块将不会显示。

可使用布尔参数 {{always}} 防止该行为。
当其为正值时，模块内容仍会显示，但除 {{%%avatar%%}}（默认头像链接）外，其它变量将不可用。

因此，可以使用以下方式检测用户是否已登录：

[[code]]
[[module ListUsers always="yes"]]
  [[if %%number%%]]
    用户已登录
  [[else]]
    用户未登录
  [[/if]]
[[/module]]
[[/code]]

++ 模块 Redirect

: 支持参数 : ✅
: 支持内容 : ❌

将当前页面重定向至另一页面。支持参数：

* {{destination}} — 目标地址。
不会进行标准过滤，仅禁止以 {{data:}} 或 {{javascript:}} 开头的链接。

* {{noredirect}} — 当为正的 [#booleans 布尔值] 时，阻止模块在当前页面生效。
通常在编辑包含该模块的页面时使用（{{/noredirect/true}}）。

示例：

[[code]]
[[module Redirect destination="/scp-1730"]]
[[/code]]

++ 模块 InterWiki

: 支持参数 : ✅
: 支持内容 : ✅

系统模块。用于在侧边栏显示当前页面的多语言翻译。

通过 API [https://crom.avn.sh/ Crom] 实现，由英文 SCP 社区创建并维护。

对于每个找到的翻译，模块内容会被复制并作为标记处理，并进行 [#autoreplace 自动替换]，可使用以下变量：

* {{%%url%%}} — 翻译页面地址。
* {{%%language%%}} — 翻译语言名称或对应维基名称（若同一语言有多个维基），语言由 {{language}} 参数指定。
* {{%%language_native%%}} — 翻译语言在其自身语言中的名称。
* {{%%language_code%%}} — 翻译语言代码（例如 {{en}}, {{ru}}）。

InterWiki 模块参数：

* {{article}} — 要获取翻译列表的文章名称。

* {{prependLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染前输出。
不支持变量。

* {{appendLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染后输出。
不支持变量。

* {{language}} — {{%%language%%}} 所使用的显示语言。

* {{order}} — 排序方向与变量。可使用 {{url}}, {{language}}, {{language_native}}, {{language_code}}。
可添加 {{desc}} 表示降序（例如 {{order="language_native desc"}}）。

* {{omitlanguage}} — 在获取翻译列表时忽略的语言（通常为当前维基语言）。

* {{empty}} — 若未找到任何翻译时显示的标记。

* {{loading}} — 页面加载完成后立即显示，在获取翻译数据之前显示的标记。

++ 模块 TagCloud 与 PagesByTag

系统模块，用于支持页面 <<[[[system:page-tags|标签云]]]>> 的功能。

++ 模块 SiteChanges

系统模块，用于支持页面 <<[[[system:recent-changes|最近更改]]]>> 的功能。

++ 模块 ForumStart、ForumCategory、ForumThread、ForumNewThread、ForumNewPost

系统模块，用于支持论坛功能。

* ForumStart 模块必须位于系统页面 {{forum:start}}，用于显示分区列表。
* ForumCategory 模块必须位于系统页面 {{forum:category}}，用于显示指定分区的主题列表。
* ForumThread 模块必须位于系统页面 {{forum:thread}}，用于显示指定主题中的帖子。
* ForumNewThread 模块必须位于系统页面 {{forum:new-thread}}，用于在指定分区创建新主题。
* ForumNewPost 不直接用于页面，而通过主题页面的模块 API 使用。

++ 模块 RecentPosts

系统模块，用于支持页面 <<[[[forum:recent-posts|论坛最新帖子]]]>> 的功能。

[[/div]]
', '''-0'':715,2104 ''/a'':646,652,1297,2035,2041,2686 ''/code'':558,649,655,667,803,844,861,880,896,905,919,1014,1035,1056,1058,1060,1366,1947,2038,2044,2056,2192,2233,2250,2269,2285,2294,2308,2403,2424,2445,2447,2449,2755 ''/div'':263,304,321,322,331,332,342,556,614,666,816,821,826,1001,1013,1652,1693,1710,1711,1720,1721,1731,1945,2003,2055,2205,2210,2215,2390,2402 ''/files.projwikit.unitreaty.org/local'':1373,2762 ''/li'':725,752,763,774,776,786,2114,2141,2152,2163,2165,2175 ''/module'':213,1032,1033,1602,2421,2422 ''/noinclude'':341,1730 ''/other_page/param'':461,1850 ''/page1/param/example1/param2/example2@@'':410,1799 ''/span'':585,612,895,1974,2001,2284 ''/tags/ref_attributes.asp'':1111,2500 ''/ul'':775,787,2164,2176 ''0'':99,157,681,1238,1488,1546,2070,2627 ''0.5'':682,2071 ''050'':65,178,180,1454,1567,1569 ''1'':1233,1378,2622,2767 ''127'':680,2069 ''16px'':57,1446 ''1px'':35,117,153,176,1424,1506,1542,1565 ''2'':1050,1059,2439,2448 ''25'':450,454,455,1839,1843,1844 ''255'':679,2068 ''25path'':451,1840 ''32px'':41,131,135,1430,1520,1524 ''4px'':82,86,1471,1475 ''500'':69,1458 ''700px'':201,1590 ''7cparam'':453,1842 ''8px'':31,93,123,149,1420,1482,1512,1538 ''aaa'':37,1426 ''abc'':601,1990 ''accept'':1149,2538 ''actual'':168,186,217,1557,1575,1606 ''actual-page-cont'':167,185,216,1556,1574,1605 ''align'':165,1150,1554,2539 ''allowfullscreen'':1243,2632 ''allowpaymentrequest'':1244,2633 ''alt'':1120,2509 ''anchor'':932,936,1282,2321,2325,2671 ''async'':1245,2634 ''attribut'':1089,1286,1322,1339,2478,2675,2711,2728 ''autocapit'':1151,2540 ''autofocus'':1246,2635 ''autoplay'':1152,1247,2541,2636 ''autoreplac'':249,1638 ''avatar'':1031,2420 ''b'':611,647,653,664,802,1052,1057,1329,2000,2036,2042,2053,2191,2441,2446,2718 ''background'':62,136,1029,1153,1451,1525,2418,2542 ''bbcode'':242,1631 ''bgcolor'':1154,2543 ''blank'':1300,2689 ''block'':508,522,1897,1911 ''blockquot'':516,863,992,1312,1905,2252,2381,2701 ''bodi'':1028,2417 ''bold'':1334,2723 ''boolean'':1226,2615 ''border'':33,84,116,121,151,174,193,1155,1422,1473,1505,1510,1540,1563,1582,2544 ''border-bottom'':32,150,173,1421,1539,1562 ''border-bottom-styl'':192,1581 ''border-radius'':83,120,1472,1509 ''bottom'':30,34,134,152,175,194,1419,1423,1523,1541,1564,1583 ''break'':209,211,1598,1600 ''break-al'':210,1599 ''buffer'':1156,2545 ''c'':613,648,654,665,2002,2037,2043,2054 ''cascadia'':73,1462 ''categori'':742,745,754,757,765,769,2131,2134,2143,2146,2154,2158 ''char'':1342,1356,2731,2745 ''charact'':1348,2737 ''check'':1157,1248,2546,2637 ''cite'':1158,2547 ''class'':215,255,301,309,315,325,328,337,603,607,610,1122,1604,1644,1690,1698,1704,1714,1717,1726,1992,1996,1999,2511 ''clear'':42,814,819,824,1431,2203,2208,2213 ''code'':58,59,60,80,91,94,96,202,256,302,310,316,326,329,338,484,552,604,645,651,661,800,841,856,874,889,898,913,1010,1022,1038,1045,1047,1049,1051,1357,1364,1374,1447,1448,1449,1469,1480,1483,1485,1591,1645,1691,1699,1705,1715,1718,1727,1873,1941,1993,2034,2040,2050,2189,2230,2245,2263,2278,2287,2302,2399,2411,2427,2434,2436,2438,2440,2746,2753,2763 ''col'':1159,2548 ''collaps'':518,1083,1907,2472 ''color'':64,179,1453,1568 ''colspan'':1124,2513 ''column'':109,1498 ''compon'':359,361,371,1748,1750,1760 ''content'':18,22,26,46,51,102,112,115,140,145,160,170,188,219,385,1407,1411,1415,1435,1440,1491,1501,1504,1529,1534,1549,1559,1577,1608,1774 ''contentedit'':1160,2549 ''control'':1161,1249,2550,2638 ''coord'':1162,2551 ''courier'':75,77,1464,1466 ''css'':8,15,524,675,1027,1384,1397,1404,1913,2064,2416,2773 ''cоde'':1055,2444 ''datetim'':1163,2552 ''dd'':142,162,1531,1551 ''decod'':1164,2553 ''decor'':183,1572 ''default'':366,1165,1250,1755,2554,2639 ''dir'':1166,2555 ''dirnam'':1167,2556 ''disabl'':1168,1251,2557,2640 ''display'':104,502,505,521,1493,1891,1894,1910 ''div'':214,254,300,308,314,324,327,336,515,534,547,553,560,602,662,812,817,822,990,997,1011,1603,1643,1689,1697,1703,1713,1716,1725,1904,1923,1936,1942,1949,1991,2051,2201,2206,2211,2379,2386,2400 ''dl'':103,141,146,161,1492,1530,1535,1550 ''dot'':177,1566 ''download'':1169,2558 ''draggabl'':1170,2559 ''dt'':147,1536 ''eee'':119,155,1508,1544 ''expr'':392,421,429,438,1781,1810,1818,1827 ''f'':1237,2626 ''f7f7f7'':63,1452 ''fals'':1235,1236,2624,2625 ''famili'':72,1461 ''ffe'':694,697,2083,2086 ''float'':124,1513 ''font'':67,71,1456,1460 ''font-famili'':70,1459 ''font-weight'':66,1455 ''footnoteblock'':517,1906 ''form'':1172,2561 ''formnovalid'':1252,2641 ''framebord'':1225,2614 ''freemono'':78,1467 ''googl'':1296,2685 ''google.com'':1295,2684 ''grid'':105,107,1494,1496 ''grid-template-column'':106,1495 ''h'':959,963,2348,2352 ''h1'':19,47,1408,1436 ''h2'':23,48,52,1412,1437,1441 ''h3'':27,53,1416,1442 ''handl'':1310,2699 ''header'':827,1173,2216,2562 ''height'':1174,2563 ''hidden'':127,1175,1253,1516,2564,2642 ''high'':1176,2565 ''hover'':191,1580 ''href'':172,190,1126,1294,1561,1579,2515,2683 ''hreflang'':1177,2566 ''html'':6,240,528,971,1040,1064,1074,1077,1088,1091,1099,1102,1105,1113,1241,1285,1287,1321,1323,1338,1340,1343,1351,1381,1395,1629,1917,2360,2429,2453,2463,2466,2477,2480,2488,2491,2494,2502,2630,2674,2676,2710,2712,2727,2729,2732,2740,2770 ''html-attribut'':1087,1284,1320,1337,2476,2673,2709,2726 ''https'':1372,2761 ''id'':1128,1136,1139,2517,2525,2528 ''iftag'':991,2380 ''imag'':497,1886 ''includ'':228,251,257,272,279,294,306,311,318,1617,1640,1646,1661,1668,1683,1695,1700,1707 ''inlin'':503,507,1892,1896 ''inline-block'':506,1895 ''input'':433,1822 ''inputmod'':1178,2567 ''ismap'':1179,1254,2568,2643 ''itemprop'':1180,2569 ''itemscop'':1255,2644 ''javascript'':1382,2771 ''jewalki'':477,1866 ''json'':424,1813 ''kind'':1181,2570 ''label'':1182,2571 ''lang'':1183,2572 ''left'':130,820,1519,2209 ''li'':703,726,741,753,764,777,2092,2115,2130,2142,2153,2166 ''link'':700,1309,2089,2698 ''link-handl'':1308,2697 ''list'':1184,2573 ''listpag'':379,380,1768,1769 ''listus'':1025,2414 ''liter'':482,784,946,1871,2173,2335 ''loop'':1185,1256,2574,2645 ''low'':1186,2575 ''lu'':1024,1034,2413,2423 ''margin'':39,55,98,129,133,156,1428,1444,1487,1518,1522,1545 ''margin-bottom'':132,1521 ''margin-left'':128,1517 ''margin-top'':38,54,1427,1443 ''markup'':3,1392 ''max'':111,114,199,1187,1500,1503,1588,2576 ''max-cont'':110,113,1499,1502 ''max-width'':198,1587 ''maxlength'':1188,2577 ''media'':197,1586 ''mime'':1387,2776 ''min'':1189,2578 ''minlength'':1190,2579 ''modul'':14,378,458,485,1023,1026,1039,1403,1767,1847,1874,2412,2415,2428 ''module-listpag'':377,1766 ''mono'':74,1463 ''monospac'':79,1468 ''multipl'':1191,1257,2580,2646 ''mute'':1192,1258,2581,2647 ''myid'':1137,1142,2526,2531 ''name'':1193,2582 ''new'':76,1465 ''noinclud'':333,339,348,351,1722,1728,1737,1740 ''nomodul'':1259,2648 ''none'':184,1573 ''novalid'':1260,2649 ''nowrap'':90,1479 ''open'':1261,2650 ''optimum'':1194,2583 ''outer'':1048,1061,2437,2450 ''overflow'':126,1515 ''p'':95,549,1484,1938 ''pad'':29,81,92,148,1418,1470,1481,1537 ''padding-bottom'':28,1417 ''page'':17,21,25,45,50,101,139,144,159,169,187,218,1406,1410,1414,1434,1439,1490,1528,1533,1548,1558,1576,1607 ''page-cont'':16,20,24,44,49,100,138,143,158,1405,1409,1413,1433,1438,1489,1527,1532,1547 ''page1'':298,312,319,402,743,746,755,758,766,770,1687,1701,1708,1791,2132,2135,2144,2147,2155,2159 ''param'':303,313,320,389,404,414,419,422,430,439,444,464,960,964,966,1692,1702,1709,1778,1793,1803,1808,1811,1819,1828,1833,1853,2349,2353,2355 ''param2'':406,1795 ''pat'':958,962,2347,2351 ''path'':388,390,391,393,413,418,420,428,437,442,462,1777,1779,1780,1782,1802,1807,1809,1817,1826,1831,1851 ''path-param'':387,1776 ''pattern'':1195,2584 ''placehold'':1196,2585 ''playsinlin'':1262,2651 ''poster'':1197,2586 ''pre'':61,97,1450,1486 ''preload'':1198,2587 ''projwikit.unitreaty.org'':409,1798 ''projwikit.unitreaty.org/page1/param/example1/param2/example2@@'':408,1797 ''quot'':1318,2707 ''radius'':85,122,1474,1511 ''readon'':1199,1263,2588,2652 ''red'':671,1116,1119,1121,1123,1125,1127,1143,1145,1147,2060,2505,2508,2510,2512,2514,2516,2532,2534,2536 ''redirect'':459,1848 ''requir'':1200,1264,2589,2653 ''revers'':1201,1265,2590,2654 ''rgb'':688,2077 ''rgba'':678,689,2067,2078 ''right'':125,166,825,1514,1555,2214 ''role'':1202,2591 ''row'':1203,2592 ''rowspan'':1144,2533 ''rrggbb'':686,2075 ''rrggbbaa'':687,2076 ''scope'':1204,2593 ''scroll'':1224,2613 ''select'':1205,1266,2594,2655 ''shape'':1206,2595 ''show'':1084,2473 ''size'':1207,1208,2596,2597 ''solid'':36,118,154,196,1425,1507,1543,1585 ''some-class'':608,1997 ''space'':89,205,1478,1594 ''span'':495,584,606,886,892,908,923,1096,1209,1884,1973,1995,2275,2281,2297,2312,2485,2598 ''spellcheck'':1210,2599 ''src'':1211,2600 ''srclang'':1212,2601 ''srcset'':1213,2602 ''start'':1214,2603 ''step'':1215,2604 ''strong'':1335,2724 ''style'':195,813,818,823,1146,1584,2202,2207,2212,2535 ''tabindex'':1216,2605 ''tabl'':542,929,1931,2318 ''target'':1148,1299,1305,2537,2688,2694 ''templat'':108,360,362,367,368,1497,1749,1751,1756,1757 ''text'':164,182,317,330,435,1012,1553,1571,1706,1719,1824,2401 ''text-align'':163,1552 ''text-decor'':181,1570 ''titl'':1217,2606 ''toc'':13,514,714,1402,1903,2103 ''top'':40,56,1429,1445 ''translat'':1218,2607 ''true'':1230,1231,2619,2620 ''truespe'':1267,2656 ''type'':434,1219,1823,2608 ''u'':1131,1141,2520,2530 ''u-myid'':1140,2529 ''ul'':702,740,2091,2129 ''url'':394,443,446,452,463,1030,1783,1832,1835,1841,1852,2419 ''usemap'':1220,2609 ''user'':498,1887 ''valu'':436,1221,1825,2610 ''weight'':68,1457 ''white'':88,137,204,1477,1526,1593 ''white-spac'':87,203,1476,1592 ''width'':200,1222,1589,2611 ''wiki'':2,1391 ''word'':208,1597 ''word-break'':207,1596 ''wrap'':206,1223,1595,2612 ''www.w3schools.com'':1110,2499 ''www.w3schools.com/tags/ref_attributes.asp'':1109,2498 ''xml'':1383,2772 ''yes'':1234,2623 ''一般来说'':500,1889 ''上标'':629,2018 ''下一个表格单元格'':557,1946 ''下一行'':891,894,2280,2283 ''下划线'':635,2024 ''下列代码在插入文章后将显示为一行'':600,1989 ''下标'':632,2021 ''不使用'':1076,2465 ''不同于'':1073,2462 ''不必额外使用'':691,2080 ''不支持六级以上标题'':836,2225 ''不是正确语法'':641,2030 ''与'':696,1365,2085,2754 ''中'':538,543,1927,1932 ''中使用的所有变量'':382,1771 ''中打开'':724,1303,2113,2692 ''中来绕过该限制'':550,1939 ''中的代码'':299,1688 ''中访问参数'':403,1792 ''为了使该元素正常工作'':264,1653 ''为了使该元素被解析为破折号而不是删除线语法'':984,2373 ''为了在文章'':401,1790 ''为了正确生效'':807,2196 ''为了防止这些可视化元素被包含到作者文章中'':346,1735 ''为此'':594,1983 ''主要用于展示标记示例而不被立即解析'':1368,2757 ''之前'':1018,2407 ''之后也不能有任何文本'':270,1659 ''之后的数量没有限制'':853,2242 ''之间的标记规则'':1367,2756 ''也不一定与块级元素兼容'':236,1625 ''也可用于像'':967,2356 ''也可用于破坏自动替换语法'':955,2344 ''仅在行首使用时有效'':810,2199 ''从'':1377,2766 ''从其他文章插入代码'':252,1641 ''从而可以在'':1389,2778 ''从而可以通过链接跳转到该位置'':934,2323 ''代码块'':1358,2747 ''以'':423,1812 ''以下指南是关于pojectwikit网站整体运行机制以及维基标记语言'':1,1390 ''以下语法会创建两个嵌套的引用块'':855,2244 ''以及'':494,1883 ''以及一个起始标签'':995,2384 ''以及其中的各个参数'':273,1662 ''以及段落创建'':590,1979 ''任意文本'':340,1729 ''会对指定文章中的所有变量进行自动替换'':282,1671 ''会被替换为空格'':750,2139 ''但不推荐这样使用'':969,2358 ''但不能跨多个段落'':643,2032 ''但可以嵌入块级元素或显式换行'':839,2228 ''但在修饰符'':1017,2406 ''但在标记层面更加'':1326,2715 ''但在段落判定上不视为空'':582,1971 ''但实际上并不是'':939,2328 ''但该属性的实际值不会影响段落生成'':526,1915 ''作为前缀'':1132,2521 ''你也可以在代码分成多行时阻止换行'':589,1978 ''使用'':305,1694 ''使用修饰符'':1291,2680 ''使用块'':1054,2443 ''使用完整标识符'':733,2122 ''使用示例'':1355,2744 ''使用该元素时'':275,1664 ''使用说明文档以及预览内容'':345,1734 ''例如'':283,383,400,432,457,533,551,583,599,639,660,693,713,797,840,846,888,935,953,957,989,996,1000,1021,1082,1094,1135,1292,1672,1772,1789,1821,1846,1922,1940,1972,1988,2028,2049,2082,2102,2186,2229,2235,2277,2324,2342,2346,2378,2385,2389,2410,2471,2483,2524,2681 ''例如在记录该功能自身文档时'':356,1745 ''例如块属性或链接名称中'':979,2368 ''例如此处的'':370,1759 ''例如表格或列表'':806,2195 ''例如表达式'':677,2066 ''修饰符'':245,1634 ''修饰符只写在起始标签中'':1008,2397 ''值'':398,1069,1072,1787,2458,2461 ''值1'':260,1649 ''值2'':262,1651 ''值也能正确写入'':441,1830 ''允许使用的属性列表如下'':1107,2496 ''允许在原本不支持换行的元素中插入换行'':805,2194 ''允许忽略'':1363,2752 ''元素'':271,1660 ''元素1'':875,890,2264,2279 ''元素2'':876,893,2265,2282 ''元素2.1'':877,2266 ''元素2.2'':878,2267 ''元素3'':879,2268 ''元素与其内部文本之间不得有空格'':638,2027 ''元素位置控制'':788,2177 ''元素内部可以包含任何其他元素'':656,2045 ''元素可以跨多行'':642,2031 ''元素级'':290,1679 ''全宽'':1270,1315,1360,2659,2704,2749 ''全宽元素'':510,1899 ''全宽元素周围不会创建段落'':570,1959 ''其两侧必须有空格'':985,2374 ''其他'':931,2320 ''其值将会叠加'':1306,2695 ''其工作方式与'':1353,2742 ''其标记中的属性会直接插入生成的'':1101,2490 ''其格式在外观上类似于'':239,1628 ''其目标是全面'':10,1399 ''具有严格规则'':238,1627 ''具有更好的可读性'':864,2253 ''内的内容会被包裹为段落'':561,1950 ''内部不支持换行'':838,2227 ''内部使用该块的标准结束标签'':1041,2430 ''写在块名称之后'':1016,2405 ''分为两大类'':490,1879 ''分别对应一级至六级标题'':835,2224 ''分别等同于'':811,2200 ''分类前缀会被移除等'':751,2140 ''分类模板'':357,1746 ''分类模板是形如'':358,1747 ''分隔线'':512,1901 ''列表'':513,867,1902,2256 ''列表的嵌套级别由'':881,2270 ''列表项中可以通过'':884,2273 ''则下一行会'':596,1985 ''则会以不同颜色显示'':738,2127 ''则会在新窗口'':722,2111 ''则可视为行内元素'':509,1898 ''则插入'':449,1838 ''则插入文本'':417,427,1806,1816 ''则自动使用其标题作为链接文本'':761,2150 ''创建一个具有指定标识符的元素'':933,2322 ''创建指向'':744,756,768,2133,2145,2157 ''创建站内文章链接'':735,2124 ''删除线'':626,2015 ''别名'':1281,1317,1333,1347,2670,2706,2722,2736 ''到当前行'':598,1987 ''前三个'':852,2241 ''前两种可以相互嵌套'':871,2260 ''前的空格或制表符数量决定'':883,2272 ''前面从行首开始不得有任何文本'':266,1655 ''加粗文本'':1330,2719 ''包括其他块级元素'':248,1637 ''包括后续行'':793,2182 ''包括块级元素'':657,2046 ''包括普通文本'':492,1881 ''包括标题'':511,1900 ''包括空格'':267,1656 ''包裹内容来实现多行'':887,2276 ''包裹的内容被当作标记语法解析'':949,2338 ''协议'':1085,2474 ''即便指定了参数'':965,2354 ''参数'':397,1068,1071,1786,2457,2460 ''参数1'':259,285,1648,1674 ''参数2'':261,1650 ''只能用于行尾'':796,2185 ''可以使用'':347,783,921,1736,2172,2310 ''可以使用两种方法'':580,1969 ''可以使用块级元素'':928,2317 ''可以保持可读性'':592,1981 ''可以包含文本或其他块级元素的块级元素'':998,2387 ''可以通过'':384,1773 ''可以通过如下地址访问'':407,1796 ''可以通过如下格式的单独文件访问'':1371,2760 ''可以通过将所需文本包裹在'':546,1935 ''可用于在代码中添加技术性备注'':944,2333 ''可用于在出于视觉效果使用标记符号时避免歧义'':952,2341 ''可用于防止两个连续空行被合并为一个段落'':798,2187 ''可确保即便用户使用特殊字符'':440,1829 ''可选的标识符'':994,2383 ''右对齐文本'':916,2305 ''右引号'':981,2370 ''同时'':375,1764 ''同时不在视觉上拆分文本'':593,1982 ''同时也支持文本着色'':668,2057 ''同样'':268,293,1657,1682 ''同样适用于以下内置'':1240,2629 ''后必须保留一个空格'':941,2330 ''否则使用其标识符'':762,2151 ''否则标签不会生效'':354,1743 ''和'':7,405,975,1396,1794,2364 ''和其他诸如'':496,1885 ''和可编辑性'':865,2254 ''和易读'':1328,2717 ''回到第一级'':860,2249 ''因为元素的分类是在其从标记转换为'':527,1916 ''因此'':224,1613 ''因此不推荐使用此语法'':866,2255 ''因此以下语法是正确的'':1009,2398 ''因此可能跨多行发生'':480,1869 ''因此在仅支持纯文本的场景下不会生效'':978,2367 ''因此被嵌入的文章中可以包含完整或部分源代码'':292,1681 ''在任何行内元素中'':540,1929 ''在使用块级元素的情况下'':658,2047 ''在元素定义中'':940,2329 ''在全宽块级元素中'':531,1920 ''在功能上等同于使用'':1325,2714 ''在块级表格'':541,1930 ''在大多数自由语法元素中'':544,1933 ''在实际使用中可以用以下字符串表示'':1229,2618 ''在属性值中使用特殊字符时'':1075,2464 ''在插入之前'':281,1670 ''在文本中插入一个'':1350,2739 ''在文章开始处理之前执行'':223,1612 ''在本网站语境中最常用的属性已用'':1115,2504 ''在术语或定义中同样可以使用'':906,2295 ''在标题文本前可以添加符号'':845,2234 ''在此示例中'':559,1948 ''在段落内部插入全宽元素会在该处终止当前段落'':571,1960 ''在简单引用块'':537,1926 ''在结束的'':269,1658 ''在被插入到其他页面时忽略部分代码'':334,1723 ''在该行放置任何视觉上为空的元素'':581,1970 ''在该行末尾添加符号'':586,1975 ''在起始的'':265,1654 ''在这种情况下'':927,2316 ''地址'':704,707,734,2093,2096,2123 ''地址不能包含空格'':717,2106 ''块标识符允许在接受文本内容的块'':1036,2425 ''块的标识符是可选文本'':1015,2404 ''块级元素'':237,986,1626,2375 ''块级元素具有名称'':243,1632 ''基础知识的读者'':9,1398 ''大多数块可以以某种形式接受属性'':1062,2451 ''大致对应'':520,1909 ''如'':1037,1079,2426,2468 ''如下所示'':799,2188 ''如果为某个分类'':369,1758 ''如果你希望空行仅作为空行'':578,1967 ''如果在不创建段落的元素中存在文本'':573,1962 ''如果未为其指定修饰符'':532,1921 ''如果未指定此前缀'':1133,2522 ''如果未指定该变量'':416,1805 ''如果目标文章不存在'':737,2126 ''如果该元素默认显示为'':501,1890 ''如果该文章存在'':760,2149 ''如果链接以星号'':720,2109 ''如需避免'':782,2171 ''始终会显示为文本'':961,2350 ''始终生成一行文本'':950,2339 ''字典列表定义如下'':897,2286 ''字符'':1344,2733 ''字符串格式插入变量值'':425,1814 ''字面量'':483,785,951,1872,2174,2340 ''它不受此限制'':930,2319 ''定义'':900,902,904,2289,2291,2293 ''定义一段在最终页面显示时不会呈现的源代码区域'':943,2332 ''实体'':1078,2467 ''实体字符'':1352,2741 ''实体符号'':972,2361 ''实现'':1007,2396 ''对于'':1380,2769 ''对于主分类'':364,1753 ''对其所包含元素的类型不作限制'':247,1636 ''对齐'':1268,2657 ''将会被转换为'':1138,2527 ''将内部文本两端对齐'':1276,2665 ''将内部文本右对齐'':1274,2663 ''将内部文本居中对齐'':1275,2664 ''将内部文本左对齐'':1273,2662 ''将设置对应的'':1386,2775 ''尽管上文提及'':523,1912 ''居中文本'':917,2306 ''属性'':244,525,1065,1092,1242,1288,1324,1341,1633,1914,2454,2481,2631,2677,2713,2730 ''属性写法为'':1067,2456 ''属性都允许在标记中使用'':1106,2495 ''左引号'':980,2369 ''布尔属性'':1227,2616 ''干净'':1327,2716 ''并且绝不会转换为段落'':588,1977 ''并使用指定文本作为链接名称'':772,2161 ''并在该全宽元素之后创建新段落'':572,1961 ''并将该源代码插入到'':278,1667 ''并解释为什么某些内容会以目前这种方式运行'':12,1401 ''并非所有'':1104,2493 ''并非链接用途的单个方括号'':954,2343 ''开头'':721,2110 ''开始'':1379,2768 ''引用'':854,2243 ''引用块'':1313,2702 ''引言'':220,1609 ''当不希望发生自动替换时'':956,2345 ''当使用十六进制值表示颜色'':685,2074 ''形如'':284,779,1673,2168 ''必须放在段落开头使用'':791,2180 ''忧郁'':1086,2475 ''我就知道你会看到这里'':945,2334 ''或'':504,548,706,728,730,882,907,922,1070,1893,1937,2095,2117,2119,2271,2296,2311,2459 ''或使用'':885,2274 ''或在站内文章之间使用参数进行链接'':711,2100 ''或在该块内部写出'':1044,2433 ''或多个符号'':973,2362 ''或用于锚点链接'':712,2101 ''或者其上方存在一个全宽元素'':568,1957 ''或者其上方至少有一整行空行'':567,1956 ''所在的位置'':280,1669 ''所有上述文本格式化方式都遵循相同的规则'':637,2026 ''所有块级元素都遵循类似规则构建'':987,2376 ''所有文本格式元素'':493,1882 ''指向第一个标题的链接'':716,2105 ''指定'':1020,2409 ''指定了模板'':372,1761 ''按既定规则自动进行'':231,1620 ''换行通过'':1006,2395 ''排版符号的自动替换'':465,1854 ''控制换行'':577,1966 ''插入'':445,1834 ''插入指定的'':970,2359 ''支持'':1283,1289,1290,1319,1336,2672,2678,2679,2708,2725 ''支持属性'':1272,1349,1362,2661,2738,2751 ''支持的颜色'':676,2065 ''文本'':466,468,469,471,472,474,617,619,620,622,623,625,627,628,630,631,633,634,636,640,670,673,683,695,698,705,708,767,789,829,830,831,832,833,834,847,947,1855,1857,1858,1860,1861,1863,2006,2008,2009,2011,2012,2014,2016,2017,2019,2020,2022,2023,2025,2029,2059,2062,2072,2084,2087,2094,2097,2156,2178,2218,2219,2220,2221,2222,2223,2236,2336 ''文本中的空行将被视为普通换行'':575,1964 ''文本会直接放入块中'':1005,2394 ''文本格式'':616,2005 ''文档'':1114,2503 ''文档中标记为可接受布尔值的属性'':1228,2617 ''文章'':297,727,729,731,1686,2116,2118,2120 ''文章名称'':258,1647 ''斜体'':621,2010 ''无序列表中嵌套有序列表'':873,2262 ''无序列表和字典列表'':870,2259 ''无论是起始还是结束的'':350,1739 ''时'':690,2079 ''是'':1098,2487 ''是合法的'':684,2073 ''显式换行'':795,2184 ''显式换行符必须与前面的文本或元素之间用空格分隔'':808,2197 ''普通链接'':709,2098 ''暂时移除'':476,1865 ''更多详情请参阅'':1108,2497 ''替换为'':467,470,473,1856,1859,1862 ''替换为符号'':475,1864 ''有序列表'':869,2258 ''有颜色的'':672,2061 ''本指南面向具备最低限度'':5,1394 ''术语1'':899,2288 ''术语2'':901,2290 ''术语3'':903,2292 ''某些元素'':1093,2482 ''某些块可以使用修饰符'':1002,2391 ''某些站点组件同时包含可调用代码'':343,1732 ''标准'':1090,2479 ''标准过滤'':1311,2700 ''标出'':1118,2507 ''标签'':349,352,1738,1741 ''标签页'':723,1302,2112,2691 ''标记或'':241,1630 ''标记语言分为四种类型'':221,1610 ''标识符必须以'':1130,2519 ''标题'':828,914,2217,2303 ''标题2'':915,2304 ''根据上述任一条件'':574,1963 ''模块'':381,1770 ''模板中支持'':376,1765 ''横向跨越两列的单元格'':918,2307 ''正确'':644,2033 ''此后整个段落'':792,2181 ''此外'':804,2193 ''此时该标题不会被加入目录'':848,2237 ''此类链接会显示在文章的反向链接中'':736,2125 ''段落'':1271,1316,1361,2660,2705,2750 ''段落不会被创建'':539,1928 ''段落会被创建'':530,1919 ''段落划分'':230,488,1619,1877 ''段落居中'':790,2179 ''段落限制将被解除'':659,2048 ''每个块级元素都有名称'':988,2377 ''每个标记元素可能以完全不可预测的方式被解析'':234,1623 ''水平线'':849,2238 ''没有严格的格式'':233,1622 ''注释'':942,2331 ''添加水平分隔线'':850,2239 ''清除浮动元素'':809,2198 ''甚至包括在'':481,1870 ''用于阻止在块内自动创建段落'':1004,2393 ''由于'':862,2251 ''由于插入源代码是在'':287,1676 ''由于这些变量替换属于自动替换'':411,1800 ''由于这些替换不属于自动替换'':977,2366 ''由于这些符号替换发生在自动替换阶段'':479,1868 ''的'':1112,2501 ''的初始阶段完成的'':529,1918 ''的单元格'':926,2315 ''的参数中也可以包含完整或部分源代码'':295,1684 ''的变量将被替换为在该元素中指定的对应参数值'':286,1675 ''的文本会自动转换为对应链接'':780,2169 ''的文章中的代码'':307,1696 ''的替换'':976,2365 ''的直接接口'':1100,2489 ''的示例'':1046,2435 ''的语法在页面地址中传递参数'':399,1788 ''的链接'':747,759,771,2136,2148,2160 ''的隐藏页面'':363,1752 ''目标文章可以通过三种方式访问参数'':412,1801 ''直接将变量值插入文章代码'':415,1804 ''相关内容的技术文档'':4,1393 ''示例'':296,872,1053,1685,2261,2442 ''站点引擎支持通过形如'':396,1785 ''章节'':229,1618 ''符号'':692,749,974,2081,2138,2363 ''第一级'':857,2246 ''第一行'':554,842,1943,2231 ''第二级'':858,2247 ''第二级的第二行'':859,2248 ''第二行'':555,843,1944,2232 ''等'':519,993,1097,1908,2382,2486 ''等元素'':499,1888 ''等内容中'':486,1875 ''等同于使用'':1298,2687 ''等宽文本'':624,2013 ''等效'':699,2088 ''简化'':910,2299 ''类型'':1269,1279,1314,1331,1345,1359,1388,2658,2668,2703,2720,2734,2748,2777 ''粗体'':618,2007 ''粘连'':597,1986 ''系统中所有可以包含其他元素的元素'':489,1878 ''系统会自动添加'':1134,2523 ''系统会访问指定的站点文章'':276,1665 ''红色'':1117,2506 ''纵向合并'':925,2314 ''组件本身'':344,1733 ''结果'':323,1712 ''编码格式的变量值'':447,1836 ''网站支持三种列表类型'':868,2257 ''而下一个表格单元格则会被直接作为文本添加'':562,1951 ''而不会真正关闭它'':1042,2431 ''而不是'':576,1965 ''而不是创建段落'':579,1968 ''而不是文章的实际代码'':374,1763 ''而是通过'':1080,2469 ''而链接文本可以包含空格'':718,2107 ''而非'':289,1678 ''自动替换'':222,250,1611,1639 ''自动替换允许在文章中添加新的代码'':225,1614 ''自动链接'':778,2167 ''自由'':911,2300 ''自由语法'':232,615,1621,2004 ''若同时使用该修饰符和'':1304,2693 ''若未指定'':426,448,1815,1837 ''获取其源代码'':277,1666 ''获取原始文章代码'':386,1775 ''行内'':1280,1332,1346,2669,2721,2735 ''行内元素'':491,1880 ''行级'':288,1677 ''表格'':909,2298 ''要么是'':1063,2452 ''要么是特定块自定义属性'':1066,2455 ''要在单元格中添加多行文本'':920,2309 ''要在支持段落的元素中创建或分隔新段落'':564,1953 ''该修饰符写在块名称之后'':1003,2392 ''该修饰符并非对所有块级元素都可用'':535,1924 ''该元素也可用于代码高亮'':1369,2758 ''该元素只能在新行使用'':837,851,2226,2240 ''该元素在视觉上类似于块级元素'':938,2327 ''该元素无法创建跨越多行'':924,2313 ''该属性会被特殊处理'':1129,2518 ''该技巧同样适用于块级表格'':563,1952 ''该符号会被明确解释为换行'':587,1976 ''该行必须仅包含行内元素'':569,1958 ''该行必须是元素中的第一行'':566,1955 ''该链接会在新窗口'':1301,2690 ''详尽地描述所有可用功能'':11,1400 ''详见'':227,1616 ''详见各元素说明'':536,1925 ''语法'':253,335,1642,1724 ''语法几乎相同'':1354,2743 ''语法变体'':739,2128 ''语法的表格如下所示'':912,2301 ''语言'':1385,2774 ''请务必注意'':487,1876 ''请在行末添加'':595,1984 ''跳转到锚点'':937,2326 ''转义'':1081,2470 ''还必须有与起始标签对应名称的结束标签'':999,2388 ''这些代码随后会作为语法被处理'':226,1615 ''这些元素不总是彼此兼容'':235,1624 ''这允许在块级元素属性中传递包含特殊字符的复杂值'':431,1820 ''这允许在链接或传递给其他页面的参数中使用这些值'':456,1845 ''这在编写复杂代码时非常有用'':591,1980 ''这样可以将多个模块相互嵌套'':1043,2432 ''这样设计是为了降低误触发或错误触发的概率'':355,1744 ''这种自动替换可能会引发问题并破坏其他语法'':781,2170 ''进行的'':291,1680 ''通常用于外部链接'':710,2099 ''通过'':1019,2408 ''通过该元素创建的链接会经过'':1307,2696 ''通过该元素添加到页面的代码'':1370,2759 ''那么该模板将会为该分类下的所有文章渲染显示'':373,1762 ''那些原则上可以包含其他元素的块级元素'':246,1635 ''那样创建空行'':968,2357 ''都会居中对齐'':794,2183 ''都可以占用多行'':274,1663 ''都必须单独占据一整行'':353,1742 ''链接'':701,1278,2090,2667 ''链接文本'':732,2121 ''链接文本不得跨多行'':719,2108 ''链接文本同样不得跨多行'':773,2162 ''链接文本大致对应文章标识符'':748,2137 ''错误'':650,2039 ''长破折号'':982,2371 ''阻止'':948,2337 ''除外'':545,1934 ''需要注意的是'':478,983,1867,2372 ''需要满足以下多个条件'':565,1954 ''页面中'':1103,2492 ''页面中的代码块编号'':1376,2765 ''页面参数'':395,1784 ''页面名称'':1375,2764 ''页面名称为'':365,1754 ''颜色'':669,2058 ''颜色可以使用任何'':674,2063', 118);
INSERT INTO public.web_articlesearchindex VALUES (32, 'unrated source
[[[probe:no-such-one|first]]] [[[probe:no-such-two]]] [[[wanted:alpha|alpha]]]', 'unrated source
[[[probe:no-such-one|first]]] [[[probe:no-such-two]]] [[[wanted:alpha|alpha]]]', '''alpha'':15,16,31,32 ''first'':8,24 ''no-such-on'':4,20 ''no-such-two'':10,26 ''one'':7,23 ''probe'':3,9,19,25 ''sourc'':2,18 ''two'':13,29 ''unrat'':1,17 ''want'':14,30', 135);
INSERT INTO public.web_articlesearchindex VALUES (33, '[[module search]]', '[[module search]]', '''modul'':1,3 ''search'':2,4', 119);
INSERT INTO public.web_articlesearchindex VALUES (34, '[[module CSS head="true"]]
.a { color: #ff0000 }
[[/module]]
head styled body', '[[module CSS head="true"]]
.a { color: #ff0000 }
[[/module]]
head styled body', '''/module'':8,19 ''bodi'':11,22 ''color'':6,17 ''css'':2,13 ''ff0000'':7,18 ''head'':3,9,14,20 ''modul'':1,12 ''style'':10,21 ''true'':4,15', 197);
INSERT INTO public.web_articlesearchindex VALUES (35, '[[module RecentPosts]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '[[module RecentPosts]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '''modul'':1,5 ''recentpost'':2,6 ''如果您希望论坛正常工作'':3,7 ''请不要更改此页面'':4,8', 120);
INSERT INTO public.web_articlesearchindex VALUES (36, '[[module ListPages category="probe" order="name" separate="no" prependline="||~ page||" appendline="end"]]
||%%name%%||
[[/module]]', '[[module ListPages category="probe" order="name" separate="no" prependline="||~ page||" appendline="end"]]
||%%name%%||
[[/module]]', '''/module'':14,28 ''appendlin'':11,25 ''categori'':3,17 ''end'':12,26 ''listpag'':2,16 ''modul'':1,15 ''name'':6,13,20,27 ''order'':5,19 ''page'':10,24 ''prependlin'':9,23 ''probe'':4,18 ''separ'':7,21', 128);
INSERT INTO public.web_articlesearchindex VALUES (37, '[[module ForumNewThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '[[module ForumNewThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', '''forumnewthread'':2,6 ''modul'':1,5 ''如果您希望论坛正常工作'':3,7 ''请不要更改此页面'':4,8', 121);
INSERT INTO public.web_articlesearchindex VALUES (38, '[[module ListPages category="probe" order="name" name="@url|probe*"]]
%%name%%
[[/module]]', '[[module ListPages category="probe" order="name" name="@url|probe*"]]
%%name%%
[[/module]]', '''/module'':11,22 ''categori'':3,14 ''listpag'':2,13 ''modul'':1,12 ''name'':6,7,10,17,18,21 ''order'':5,16 ''probe'':4,9,15,20 ''url'':8,19', 133);
INSERT INTO public.web_articlesearchindex VALUES (39, '[[module Rate]]', '[[module Rate]]', '''modul'':1,3 ''rate'':2,4', 139);
INSERT INTO public.web_articlesearchindex VALUES (40, '[[module NoSuchModule]]', '[[module NoSuchModule]]', '''modul'':1,3 ''nosuchmodul'':2,4', 117);
INSERT INTO public.web_articlesearchindex VALUES (41, 'half source', 'half source', '''half'':1,3 ''sourc'':2,4', 15);
INSERT INTO public.web_articlesearchindex VALUES (42, '[[module ListPages category="*" order="votes desc" perPage="5" rating=">-10"]]
%%fullname%% %%rating%% %%rating_votes%% %%popularity%%
[[/module]]', '[[module ListPages category="*" order="votes desc" perPage="5" rating=">-10"]]
%%fullname%% %%rating%% %%rating_votes%% %%popularity%%
[[/module]]', '''-10'':10,26 ''/module'':16,32 ''5'':8,24 ''categori'':3,19 ''desc'':6,22 ''fullnam'':11,27 ''listpag'':2,18 ''modul'':1,17 ''order'':4,20 ''perpag'':7,23 ''popular'':15,31 ''rate'':9,12,13,25,28,29 ''vote'':5,14,21,30', 134);
INSERT INTO public.web_articlesearchindex VALUES (43, 'full source
second line', 'full source
second line', '''full'':1,5 ''line'':4,8 ''second'':3,7 ''sourc'':2,6', 12);
INSERT INTO public.web_articlesearchindex VALUES (44, 'host=%%this|title%% full=%%this|fullname%% own=%%title%% miss=%%this|nosuchvar%%', 'host=%%this|title%% full=%%this|fullname%% own=%%title%% miss=%%this|nosuchvar%%', '''full'':4,15 ''fullnam'':6,17 ''host'':1,12 ''miss'':9,20 ''nosuchvar'':11,22 ''titl'':3,8,14,19', 16);
INSERT INTO public.web_articlesearchindex VALUES (45, 'parent source', 'parent source', '''parent'':1,3 ''sourc'':2,4', 11);
INSERT INTO public.web_articlesearchindex VALUES (46, '+ pwikit 新功能演示

这一页把 2026-08-31 加进来的四个模块和几项站点设置放在一起看。

++ 一、@@[[module time]]@@ —— 当前时间

现在是 **[[module time]]%%currentyear%% 年 %%currentmonth%% 月 %%currentday%% 日 %%currenthour%%:%%currentminute%%[[/module]]**

年、月、日、时、分各自是一个 {{odate}} 元素，**由读者自己的浏览器按本地时区渲染** —— 换个时区打开这一页，
上面的数字会跟着变；没开 JS 的读者看到的是服务端那一侧的值。可用的变量：

* @@%%currentyear%%@@ 年（四位）
* @@%%currentmonth%%@@ 月（补零，{{08}} 不是 {{8}}）
* @@%%currentday%%@@ 日 ｜ @@%%currenthour%%@@ 时 ｜ @@%%currentminute%%@@ 分 ｜ @@%%currentsecond%%@@ 秒

少一个 r 的 @@%%curentyear%%@@ 也认。**这个模块是有正文的，要写 @@[[/module]]@@。**

> **已知代价**：模块在 ftml 里是块级的，所以它自成一段，句子中间嵌不进去。
> 页脚是唯一的例外 —— 那里由 Go 自己渲染并把段落剥掉。

++ 二、@@[[module members]]@@ —— 全站成员

按用户编号排序，默认每页 100 人（这里用 @@perpage="10"@@），@@%%index%%@@ 是**跨页连续**的自动编号。

||~ # ||~ 用户 ||~ 编号 ||~ 注册时间 ||
[[module members perpage="10"]]
|| %%index%% || %%members%% || %%number%% || %%time%% ||
[[/module]]

++ 三、只列某一个身分组

加 @@role="editor"@@ 就只列那个组的人，收标识符或后台里的编号。

||~ # ||~ 「成员」组 ||~ 注册时间 ||
[[module members role="editor" perpage="10"]]
|| %%index%% || %%members%% || %%time%% ||
[[/module]]

++ 四、@@[[module applicationform]]@@ —— 工单

**不需要 @@[[/module]]@@，一行写完。** 不带参数是普通工单，进后台的「用户工单」，
标题栏交给提交人自己填。**没登录时只出一句提示，不出表单。**

[[module applicationform]]

++ 五、@@title="..."@@ —— 标题由页面写死

带 @@type="membershipapply"@@ 是入组申请，进后台的「申请书」，管理员点详情可以直接决定发哪个身分组，
改成「已通过」当场就发下去。这里再加一个 @@title="入组申请"@@，**标题栏就不出现了，提交上去的标题固定是这一句**。

[[module applicationform type="membershipapply" title="入组申请"]]

++ 六、@@title="no"@@ —— 干脆不要标题

同一个参数，写成 @@yes@@／@@no@@ 就是开关，写成别的就是标题本身。这一个连标题都不提交。

[[module applicationform title="no"]]

++ 七、@@[[module membershipbypassword]]@@ —— 密码入组

也**不需要 @@[[/module]]@@**。后台**默认不启用**，关着的时候这一整块一个字都不输出 —— 不去暴露它存在。
这个演示站已经打开了它，口令是 @@letmein@@，输对了发「成员」组。左边那句默认是「输入密码」：

[[module membershipbypassword]]

加 @@label="..."@@ 就换成自己的说法：

[[module membershipbypassword label="持有邀请口令？填在这里"]]

++ 八、页脚

往下看页脚那一行 —— 它是后台里可以自己写的 wikitext，**并且只放行 @@[[module time]]@@ 这一个模块**。
在那里写 @@[[module listpages]]@@ 只会得到一个「模块不存在」的报错块。

++ 九、后台在哪

* **站点设置** —— [/-/admin/web/site/1/change/ /-/admin/web/site/1/change/]：站点图标、登录页图标、页脚、注册提示、三种身分组、密码入组
* **用户工单** —— [/-/admin/web/supportticket/ /-/admin/web/supportticket/]
* **申请书** —— [/-/admin/web/membershipapplication/ /-/admin/web/membershipapplication/]
* **角色与权限** —— [/-/admin/web/role/ /-/admin/web/role/]，新增的那一项叫「访问权限表」
', '+ pwikit 新功能演示

这一页把 2026-08-31 加进来的四个模块和几项站点设置放在一起看。

++ 一、@@[[module time]]@@ —— 当前时间

现在是 **[[module time]]%%currentyear%% 年 %%currentmonth%% 月 %%currentday%% 日 %%currenthour%%:%%currentminute%%[[/module]]**

年、月、日、时、分各自是一个 {{odate}} 元素，**由读者自己的浏览器按本地时区渲染** —— 换个时区打开这一页，
上面的数字会跟着变；没开 JS 的读者看到的是服务端那一侧的值。可用的变量：

* @@%%currentyear%%@@ 年（四位）
* @@%%currentmonth%%@@ 月（补零，{{08}} 不是 {{8}}）
* @@%%currentday%%@@ 日 ｜ @@%%currenthour%%@@ 时 ｜ @@%%currentminute%%@@ 分 ｜ @@%%currentsecond%%@@ 秒

少一个 r 的 @@%%curentyear%%@@ 也认。**这个模块是有正文的，要写 @@[[/module]]@@。**

> **已知代价**：模块在 ftml 里是块级的，所以它自成一段，句子中间嵌不进去。
> 页脚是唯一的例外 —— 那里由 Go 自己渲染并把段落剥掉。

++ 二、@@[[module members]]@@ —— 全站成员

按用户编号排序，默认每页 100 人（这里用 @@perpage="10"@@），@@%%index%%@@ 是**跨页连续**的自动编号。

||~ # ||~ 用户 ||~ 编号 ||~ 注册时间 ||
[[module members perpage="10"]]
|| %%index%% || %%members%% || %%number%% || %%time%% ||
[[/module]]

++ 三、只列某一个身分组

加 @@role="editor"@@ 就只列那个组的人，收标识符或后台里的编号。

||~ # ||~ 「成员」组 ||~ 注册时间 ||
[[module members role="editor" perpage="10"]]
|| %%index%% || %%members%% || %%time%% ||
[[/module]]

++ 四、@@[[module applicationform]]@@ —— 工单

**不需要 @@[[/module]]@@，一行写完。** 不带参数是普通工单，进后台的「用户工单」，
标题栏交给提交人自己填。**没登录时只出一句提示，不出表单。**

[[module applicationform]]

++ 五、@@title="..."@@ —— 标题由页面写死

带 @@type="membershipapply"@@ 是入组申请，进后台的「申请书」，管理员点详情可以直接决定发哪个身分组，
改成「已通过」当场就发下去。这里再加一个 @@title="入组申请"@@，**标题栏就不出现了，提交上去的标题固定是这一句**。

[[module applicationform type="membershipapply" title="入组申请"]]

++ 六、@@title="no"@@ —— 干脆不要标题

同一个参数，写成 @@yes@@／@@no@@ 就是开关，写成别的就是标题本身。这一个连标题都不提交。

[[module applicationform title="no"]]

++ 七、@@[[module membershipbypassword]]@@ —— 密码入组

也**不需要 @@[[/module]]@@**。后台**默认不启用**，关着的时候这一整块一个字都不输出 —— 不去暴露它存在。
这个演示站已经打开了它，口令是 @@letmein@@，输对了发「成员」组。左边那句默认是「输入密码」：

[[module membershipbypassword]]

加 @@label="..."@@ 就换成自己的说法：

[[module membershipbypassword label="持有邀请口令？填在这里"]]

++ 八、页脚

往下看页脚那一行 —— 它是后台里可以自己写的 wikitext，**并且只放行 @@[[module time]]@@ 这一个模块**。
在那里写 @@[[module listpages]]@@ 只会得到一个「模块不存在」的报错块。

++ 九、后台在哪

* **站点设置** —— [/-/admin/web/site/1/change/ /-/admin/web/site/1/change/]：站点图标、登录页图标、页脚、注册提示、三种身分组、密码入组
* **用户工单** —— [/-/admin/web/supportticket/ /-/admin/web/supportticket/]
* **申请书** —— [/-/admin/web/membershipapplication/ /-/admin/web/membershipapplication/]
* **角色与权限** —— [/-/admin/web/role/ /-/admin/web/role/]，新增的那一项叫「访问权限表」
', '''-08'':5,244 ''-31'':6,245 ''/admin/web/membershipapplication'':233,234,472,473 ''/admin/web/role'':236,237,475,476 ''/admin/web/site/1/change'':221,222,460,461 ''/admin/web/supportticket'':230,231,469,470 ''/module'':23,62,99,119,125,180,262,301,338,358,364,419 ''08'':44,283 ''10'':83,94,115,322,333,354 ''100'':79,318 ''2026'':4,243 ''8'':46,285 ''applicationform'':122,134,154,171,361,373,393,410 ''curentyear'':58,297 ''currentday'':19,47,258,286 ''currenthour'':21,49,260,288 ''currentminut'':22,51,261,290 ''currentmonth'':17,41,256,280 ''currentsecond'':53,292 ''currentyear'':15,38,254,277 ''editor'':104,113,343,352 ''ftml'':65,304 ''go'':71,310 ''index'':84,95,116,323,334,355 ''js'':35,274 ''label'':196,200,435,439 ''letmein'':187,426 ''listpag'':214,453 ''member'':75,92,96,111,117,314,331,335,350,356 ''membershipappli'':140,156,379,395 ''membershipbypassword'':176,194,199,415,433,438 ''modul'':9,13,74,91,110,121,133,153,170,175,193,198,209,213,248,252,313,330,349,360,372,392,409,414,432,437,448,452 ''number'':97,336 ''odat'':29,268 ''perpag'':82,93,114,321,332,353 ''pwikit'':1,240 ''r'':56,295 ''role'':103,112,342,351 ''time'':10,14,98,118,210,249,253,337,357,449 ''titl'':136,149,157,160,172,375,388,396,399,411 ''type'':139,155,378,394 ''wikitext'':207,446 ''yes'':165,404 ''一'':8,247 ''一行写完'':126,365 ''七'':174,413 ''三'':100,339 ''三种身分组'':227,466 ''上面的数字会跟着变'':33,272 ''不出表单'':132,371 ''不去暴露它存在'':184,423 ''不带参数是普通工单'':127,366 ''不是'':45,284 ''不需要'':124,179,363,418 ''九'':218,457 ''也'':178,417 ''也认'':59,298 ''二'':73,312 ''五'':135,374 ''人'':80,319 ''元素'':30,269 ''入组申请'':150,158,389,397 ''全站成员'':76,315 ''八'':203,442 ''六'':159,398 ''关着的时候这一整块一个字都不输出'':183,422 ''写成'':164,403 ''写成别的就是标题本身'':168,407 ''分'':52,291 ''分各自是一个'':28,267 ''加'':102,195,341,434 ''加进来的四个模块和几项站点设置放在一起看'':7,246 ''口令是'':186,425 ''句子中间嵌不进去'':68,307 ''只会得到一个'':215,454 ''只列某一个身分组'':101,340 ''可用的变量'':37,276 ''同一个参数'':163,402 ''后台'':181,420 ''后台在哪'':219,458 ''四'':120,359 ''四位'':40,279 ''在那里写'':212,451 ''填在这里'':202,441 ''它是后台里可以自己写的'':206,445 ''密码入组'':177,228,416,467 ''少一个'':55,294 ''就只列那个组的人'':105,344 ''就换成自己的说法'':197,436 ''就是开关'':167,406 ''工单'':123,362 ''左边那句默认是'':191,430 ''已知代价'':63,302 ''已通过'':146,385 ''带'':138,377 ''干脆不要标题'':162,401 ''年'':16,24,39,255,263,278 ''并且只放行'':208,447 ''当前时间'':11,250 ''当场就发下去'':147,386 ''往下看页脚那一行'':205,444 ''成员'':107,189,346,428 ''所以它自成一段'':67,306 ''持有邀请口令'':201,440 ''按用户编号排序'':77,316 ''换个时区打开这一页'':32,271 ''提交上去的标题固定是这一句'':152,391 ''收标识符或后台里的编号'':106,345 ''改成'':145,384 ''新功能演示'':2,241 ''新增的那一项叫'':238,477 ''日'':20,26,48,259,265,287 ''时'':27,50,266,289 ''是'':85,324 ''是入组申请'':141,380 ''月'':18,25,42,257,264,281 ''标题栏交给提交人自己填'':130,369 ''标题栏就不出现了'':151,390 ''标题由页面写死'':137,376 ''模块不存在'':216,455 ''模块在'':64,303 ''没开'':34,273 ''没登录时只出一句提示'':131,370 ''注册提示'':226,465 ''注册时间'':90,109,329,348 ''现在是'':12,251 ''用户'':88,327 ''用户工单'':129,229,368,468 ''由读者自己的浏览器按本地时区渲染'':31,270 ''申请书'':143,232,382,471 ''登录页图标'':224,463 ''的'':57,296 ''的报错块'':217,456 ''的自动编号'':87,326 ''的读者看到的是服务端那一侧的值'':36,275 ''秒'':54,293 ''站点图标'':223,462 ''站点设置'':220,459 ''管理员点详情可以直接决定发哪个身分组'':144,383 ''组'':108,190,347,429 ''编号'':89,328 ''自己渲染并把段落剥掉'':72,311 ''补零'':43,282 ''要写'':61,300 ''角色与权限'':235,474 ''访问权限表'':239,478 ''跨页连续'':86,325 ''输入密码'':192,431 ''输对了发'':188,427 ''这一个模块'':211,450 ''这一个连标题都不提交'':169,408 ''这一页把'':3,242 ''这个模块是有正文的'':60,299 ''这个演示站已经打开了它'':185,424 ''这里再加一个'':148,387 ''这里用'':81,320 ''进后台的'':128,142,367,381 ''那里由'':70,309 ''里是块级的'':66,305 ''页脚'':204,225,443,464 ''页脚是唯一的例外'':69,308 ''默认不启用'':182,421 ''默认每页'':78,317', 205);

INSERT INTO public.web_articleversion VALUES (1, '[[div class="top-bar"]]
* [[[main | 首页]]]
* [[[scp-173 | SCP-173]]]
* [[[missing-nav-target | 不存在的页面]]]
[[/div]]', NULL, '2026-08-20 07:26:18.30712+00', 1, NULL);
INSERT INTO public.web_articleversion VALUES (2, '[[div class="side-block"]]
+ 导航
* [[[main | 首页]]]
* [[[component:box | 被包含的组件]]]
[[/div]]', NULL, '2026-08-20 07:26:18.329505+00', 2, NULL);
INSERT INTO public.web_articleversion VALUES (3, '[[div class="box"]]
这是一个被 include 的组件。参数 a = %%a%%
[[/div]]', NULL, '2026-08-20 07:26:18.343888+00', 3, NULL);
INSERT INTO public.web_articleversion VALUES (4, '+ 测试站首页

//斜体// 和 **粗体**，以及一个 [[[missing-page | 红链]]]。

[[include component:box a=1]]

表达式：[[#expr 1 + 1]]，比较：[[#expr 1 == 1]]，坏掉的三角函数：[[#expr sin(0)]]

[[module ListPages separate="no" limit="5"]]
%%title%%
[[/module]]

脚注[[footnote]]这是脚注内容。[[/footnote]]', NULL, '2026-08-20 07:26:18.358621+00', 4, NULL);
INSERT INTO public.web_articleversion VALUES (5, '[[include component:box a=173]]

**项目编号：** SCP-173

**项目等级：** Euclid

+ 描述

一个测试用条目，包含 [[[main | 内链]]]、[[[another-missing | 红链]]] 和一个代码块：

[[code type="python"]]
print("hello")
[[/code]]

[[module Rate]]', NULL, '2026-08-20 07:26:18.373859+00', 5, NULL);
INSERT INTO public.web_articleversion VALUES (8, '本组件被谁包含: %%this|title%% / %%this|fullname%% / 评分 %%this|rating%%', NULL, '2026-08-23 08:47:08.327583+00', 8, NULL);
INSERT INTO public.web_articleversion VALUES (9, '[[include component:probe-var]]', NULL, '2026-08-23 08:47:08.347627+00', 9, NULL);
INSERT INTO public.web_articleversion VALUES (10, '[[include component:probe-var]]', NULL, '2026-08-23 08:47:08.368336+00', 10, NULL);
INSERT INTO public.web_articleversion VALUES (11, 'parent source', NULL, '2026-08-24 10:49:07.560892+00', 11, NULL);
INSERT INTO public.web_articleversion VALUES (12, 'full source
second line', NULL, '2026-08-24 10:49:07.587269+00', 12, NULL);
INSERT INTO public.web_articleversion VALUES (13, 'bare source', NULL, '2026-08-24 10:49:07.600989+00', 13, NULL);
INSERT INTO public.web_articleversion VALUES (14, 'rated source', NULL, '2026-08-24 10:49:07.618782+00', 14, NULL);
INSERT INTO public.web_articleversion VALUES (15, 'half source', NULL, '2026-08-24 10:50:52.698229+00', 15, NULL);
INSERT INTO public.web_articleversion VALUES (16, 'host=%%this|title%% full=%%this|fullname%% own=%%title%% miss=%%this|nosuchvar%%', NULL, '2026-08-24 10:55:13.259728+00', 16, NULL);
INSERT INTO public.web_articleversion VALUES (17, '[[include probe:included]]', NULL, '2026-08-24 10:56:38.47006+00', 17, NULL);
INSERT INTO public.web_articleversion VALUES (19, 'before
[[module Redirect destination="/probe:full"]]
after', NULL, '2026-08-26 14:00:38.164929+00', 112, NULL);
INSERT INTO public.web_articleversion VALUES (20, 'visible text
[[module PageDescription]]custom description[[/module]]', NULL, '2026-08-26 14:00:38.219712+00', 113, NULL);
INSERT INTO public.web_articleversion VALUES (21, '[[module PageImage src="probe:full/cover.png"]]body text', NULL, '2026-08-26 14:00:38.251655+00', 114, NULL);
INSERT INTO public.web_articleversion VALUES (22, '[[module PagesByTag tag="lang:en"]]', NULL, '2026-08-26 14:00:38.281141+00', 115, NULL);
INSERT INTO public.web_articleversion VALUES (23, '[[module PagesByTag tag="zeta"]]', NULL, '2026-08-26 14:00:38.311788+00', 116, NULL);
INSERT INTO public.web_articleversion VALUES (24, '[[module NoSuchModule]]', NULL, '2026-08-26 14:00:38.341921+00', 117, NULL);
INSERT INTO public.web_articleversion VALUES (25, '+ 恭喜！一切正常！', NULL, '2026-08-26 16:12:44.623389+00', 4, NULL);
INSERT INTO public.web_articleversion VALUES (26, '以下指南是关于PojectWikit网站整体运行机制以及维基标记语言（wiki markup）相关内容的技术文档。

本指南面向具备最低限度 HTML 和 CSS 基础知识的读者；其目标是全面、详尽地描述所有可用功能，并解释为什么某些内容会以目前这种方式运行。

[[toc]]

[[module CSS]]

#page-content h1, #page-content h2, #page-content h3 {
  padding-bottom: 8px;
  border-bottom: 1px solid #aaa;
  margin-top: 32px;
  clear: both;
}

#page-content h1 + h2, #page-content h2 + h3 {
  margin-top: 16px;
}

code, .code, .code pre {
  background: #f7f7f7;
  color: #050;
  font-weight: 500;
  font-family: ''Cascadia Mono'', ''Courier New'', Courier, FreeMono, monospace;
}

code {
  padding: 4px;
  border-radius: 4px;
  white-space: nowrap;
}

.code {
  padding: 8px;
}

.code p, .code pre {
  margin: 0;
}

#page-content dl {
  display: grid;
  grid-template-columns: max-content max-content;
  border: 1px solid #eee;
  border-radius: 8px;
  float: right;
  overflow: hidden;
  margin-left: 32px;
  margin-bottom: 32px;
  background: white;
}

#page-content dl dd, #page-content dl dt {
  padding: 8px;
  border-bottom: 1px solid #eee;
  margin: 0;
}

#page-content dl dd {
  text-align: right;
}

.actual-page-content a[href^="#"] {
  border-bottom: 1px dotted #050;
  color: #050;
  text-decoration: none;
}

.actual-page-content a[href^="#"]:hover {
  border-bottom-style: solid;
}
  @media (max-width: 700px) {
code {
  white-space: wrap;
  word-break: break-all;
}
  }
[[/module]]

[[div class="actual-page-content"]]

+ 引言

标记语言分为四种类型：

* **自动替换** 在文章开始处理之前执行。因此，自动替换允许在文章中添加新的代码，这些代码随后会作为语法被处理。详见 {{[[include]]}} 章节。

* **段落划分** 按既定规则自动进行。

* **自由语法** 没有严格的格式；每个标记元素可能以完全不可预测的方式被解析。这些元素不总是彼此兼容，也不一定与块级元素兼容。

* **块级元素** 具有严格规则；其格式在外观上类似于 HTML 标记或 BBCode。块级元素具有名称、属性、修饰符。那些原则上可以包含其他元素的块级元素，对其所包含元素的类型不作限制（包括其他块级元素）。

+ [[# autoreplace]] 自动替换

++ {{[[include]]}}：从其他文章插入代码

语法：

[[div class="code"]]
@@[[include 文章名称 参数1 = 值1 | 参数2 = 值2]]@@
[[/div]]

为了使该元素正常工作，在起始的 {{[[}} 前面从行首开始不得有任何文本（包括空格）。同样，在结束的 {{]]}} 之后也不能有任何文本。

元素 {{[[include]]}}，以及其中的各个参数，都可以占用多行。

使用该元素时，系统会访问指定的站点文章，获取其源代码，并将该源代码插入到 {{[[include]]}} 所在的位置。

在插入之前，会对指定文章中的所有变量进行自动替换。例如，形如 {{@@{$参数1}@@}} 的变量将被替换为在该元素中指定的对应参数值。

由于插入源代码是在“行级”而非“元素级”进行的，因此被嵌入的文章中可以包含完整或部分源代码。同样，{{[[include]]}} 的参数中也可以包含完整或部分源代码。

示例：

* 文章 {{page1}} 中的代码： _
[[div class="code"]]
@@{$param}@@
[[/div]]

* 使用 {{[[include]]}} 的文章中的代码： _
[[div class="code"]]
@@[[include page1 param=[[div class="code"]] ]]@@
@@text@@
@@[[include page1 param=[[/div]] ]]@@
[[/div]]

* 结果： _
[[div class="code"]]
@@[[div class="code"]]@@
@@text@@
@@[[/div]]@@
[[/div]]

++ {{[[noinclude]]}}：在被插入到其他页面时忽略部分代码

语法：

[[div class="code"]]
@@[[noinclude]]@@
...任意文本...
@@[[/noinclude]]@@
[[/div]]

某些站点组件同时包含可调用代码（组件本身）、使用说明文档以及预览内容。

为了防止这些可视化元素被包含到作者文章中，可以使用 {{[[noinclude]]}} 标签。

无论是起始还是结束的 {{[[noinclude]]}} 标签，都必须单独占据一整行，否则标签不会生效。这样设计是为了降低误触发或错误触发的概率，例如在记录该功能自身文档时。

++ 分类模板

分类模板是形如 [[[component:_template|component:_template]]] 的隐藏页面。对于主分类，页面名称为 [[[_default:_template|_template]]].

如果为某个分类（例如此处的 {{component}}）指定了模板，那么该模板将会为该分类下的所有文章渲染显示，**而不是文章的实际代码**。同时，模板中支持 [#module-listpages ListPages 模块] 中使用的所有变量。例如，可以通过 {{%%content%%}} 获取原始文章代码。

++ [[# path-params]] {{%%path%%}}, {{%%path_expr%%}}, {{%%path_url%%}}：页面参数

站点引擎支持通过形如 {{/参数/值}} 的语法在页面地址中传递参数。

例如，为了在文章 {{page1}} 中访问参数 {{%%param%%}} 和 {{%%param2%%}}，可以通过如下地址访问：

{{@@https://projwikit.unitreaty.org/page1/param/example1/param2/example2@@}}

由于这些变量替换属于自动替换，目标文章可以通过三种方式访问参数：

* {{%%path|param%%}} 直接将变量值插入文章代码；如果未指定该变量，则插入文本 {{%%path|param%%}}。

* {{%%path_expr|param%%}} 以 JSON 字符串格式插入变量值；若未指定，则插入文本 {{"%%path_expr|param%%"}}。这允许在块级元素属性中传递包含特殊字符的复杂值（例如 {{@@[[input type="text" value=%%path_expr|param%%]]@@}} 可确保即便用户使用特殊字符，值也能正确写入）。

* {{%%path_url|param%%}} 插入 URL 编码格式的变量值；若未指定，则插入 {{%25%25path_url%7Cparam%25%25}}。这允许在链接或传递给其他页面的参数中使用这些值（例如 {{@@[[module Redirect to="/other_page/param/%%path_url|param%%"]]@@}}）。

++ 排版符号的自动替换

* {{@<&#96;>@文本@<&#39;>@}} —— 替换为 ‘文本’。

* {{@<&#96;>@@<&#96;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 “文本”。

* {{@<&#44;>@@<&#44;>@文本@<&#39;>@@<&#39;>@}} —— 替换为 „文本”。

[!-- * {{@<&#46;>@@<&#46;>@@<&#46;>@}}, {{@<&#46;>@ @<&#46;>@ @<&#46;>@}} —— 替换为符号 "…". --] [!-- 暂时移除 // jewalky --]

需要注意的是，由于这些符号替换发生在自动替换阶段，因此可能跨多行发生，甚至包括在 [#literals 字面量]、{{@@[[code]]@@}}、{{@@[[module]]@@}} 等内容中；请务必注意。

+ 段落划分

系统中所有可以包含其他元素的元素，分为两大类：

* 行内元素。包括普通文本、所有文本格式元素，以及 {{@@[[span]]@@}} 和其他诸如 {{@@[[image]]@@}}、{{@@[[user]]@@}} 等元素。一般来说，如果该元素默认显示为 {{display: inline}} 或 {{display: inline-block}}，则可视为行内元素。

* 全宽元素。包括标题、分隔线、列表、{{@@[[toc]]@@}}、{{@@[[div]]@@}}、{{@@[[blockquote]]@@}}、{{@@[[footnoteblock]]@@}}、{{@@[[collapsible]]@@}} 等。大致对应 {{display: block}}。

尽管上文提及 CSS 属性，但该属性的实际值不会影响段落生成，因为元素的分类是在其从标记转换为 HTML 的初始阶段完成的。

段落会被创建：

* 在全宽块级元素中，如果未为其指定修饰符 {{_}}（例如 {{@@[[div_]]@@}}）。_
该修饰符并非对所有块级元素都可用，详见各元素说明。

* 在简单引用块（{{>}}）中。

段落不会被创建：

* 在任何行内元素中。

* 在块级表格（{{@@[[table]]@@}}）中。

* 在大多数自由语法元素中（{{>}} 除外）。_
可以通过将所需文本包裹在 {{@@[[div]]@@}} 或 {{@@[[p]]@@}} 中来绕过该限制，例如：_
[[code]]|| [[div]]第一行

第二行[[/div]] || 下一个表格单元格 ||[[/code]]在此示例中，{{@@[[div]]@@}} 内的内容会被包裹为段落，而下一个表格单元格则会被直接作为文本添加。_
该技巧同样适用于块级表格。

要在支持段落的元素中创建或分隔新段落，需要满足以下多个条件：

* 该行必须是元素中的第一行，或者其上方至少有一整行空行，或者其上方存在一个全宽元素。

* 该行必须仅包含行内元素。全宽元素周围不会创建段落。在段落内部插入全宽元素会在该处终止当前段落，并在该全宽元素之后创建新段落。

如果在不创建段落的元素中存在文本（根据上述任一条件），文本中的空行将被视为普通换行（{{<br>}}，而不是 {{<p>}}）。

++ 控制换行

如果你希望空行仅作为空行，而不是创建段落，可以使用两种方法：

* 在该行放置任何视觉上为空的元素（但在段落判定上不视为空）。例如 {{@@[[span]][[/span]]@@}}、{{@<@>@@<@>@@<@>@@<@>@}}、{{@<&#64;&#60;>@@<&#62;&#64;>@}}。

* 在该行末尾添加符号 {{_}}。该符号会被明确解释为换行，并且绝不会转换为段落。

你也可以在代码分成多行时阻止换行（以及段落创建）。这在编写复杂代码时非常有用，可以保持可读性，同时不在视觉上拆分文本。为此，请在行末添加 {{\}}，则下一行会“粘连”到当前行。例如，下列代码在插入文章后将显示为一行 “abc”：

[[div class="code"]]
@@a\@@
@@[[span class="some-class"]]\@@
@@b\@@
@@[[/span]]\@@
@@c@@
[[/div]]

+ 自由语法

++ 文本格式

* {{@@**文本**@@}} —— **粗体** 文本。

* {{@@//文本//@@}} —— //斜体// 文本。

* {{@@{{文本}}@@}} —— 等宽文本。

* {{@@--文本--@@}} —— --删除线-- 文本。

* {{@@^^文本^^@@}} —— ^^上标^^ 文本。

* {{@@,,文本,,@@}} —— ,,下标,, 文本。

* {{@@__文本__@@}} —— __下划线__ 文本。

所有上述文本格式化方式都遵循相同的规则：

* 元素与其内部文本之间不得有空格（例如，{{@@__ 文本 __@@}} 不是正确语法）。

* 元素可以跨多行，但不能跨多个段落。
  正确：
[[code]]//a
b
c//[[/code]]
  错误：
[[code]]//a

b

c//[[/code]]

* 元素内部可以包含任何其他元素，包括块级元素。在使用块级元素的情况下，段落限制将被解除。例如：
[[code]]//[[div]]a

b

c[[/div]]//[[/code]]

同时也支持文本着色：

* {{@@##颜色|文本##@@}} — ##red|有颜色的## 文本。
  颜色可以使用任何 CSS 支持的颜色，例如表达式 {{@@##rgba(255, 127, 0, 0.5)|文本##@@}} 是合法的。
  当使用十六进制值表示颜色（RRGGBB、RRGGBBAA、RGB、RGBA）时，不必额外使用 {{#}} 符号（例如：{{@@##ffe|文本##@@}} 与 {{@@###ffe|文本##@@}} 等效）。

++ [[# links]] 链接

[[ul]]
[[li]]{{@@[地址 文本]@@}} 或 {{[*地址 文本]}} — 普通链接。
通常用于外部链接，或在站内文章之间使用参数进行链接，或用于锚点链接（例如：{{@@[#toc-0 指向第一个标题的链接]@@}}）。
地址不能包含空格，而链接文本可以包含空格。链接文本不得跨多行。
如果链接以星号 {{*}} 开头，则会在新窗口（标签页）中打开。
[[/li]]

[[li]]{{@@[[[文章]]]@@}} 或 {{@@[[[文章|]]]@@}} 或 {{@@[[[文章|链接文本]]]@@}} — 使用完整标识符（地址）创建站内文章链接。
此类链接会显示在文章的反向链接中；如果目标文章不存在，则会以不同颜色显示。

语法变体：

[[ul]]
  [[li]]{{@@[[[category:page1]]]@@}} — 创建指向 {{category:page1}} 的链接，链接文本大致对应文章标识符（符号 {{-}} 会被替换为空格，分类前缀会被移除等）。
  [[/li]]

  [[li]]{{@@[[[category:page1|]]]@@}} — 创建指向 {{category:page1}} 的链接。
  如果该文章存在，则自动使用其标题作为链接文本；否则使用其标识符。
  [[/li]]

  [[li]]{{@@[[[category:page1|文本]]]@@}} — 创建指向 {{category:page1}} 的链接，并使用指定文本作为链接名称。
  链接文本同样不得跨多行。
  [[/li]]
[[/ul]]
[[/li]]

[[li]]自动链接：形如 {{@@http://...@@}}、{{@@ftp://...@@}} 的文本会自动转换为对应链接。
这种自动替换可能会引发问题并破坏其他语法。如需避免，可以使用 [#literals 字面量]。
[[/li]]
[[/ul]]

++ 元素位置控制

* {{@@= 文本@@}} — 段落居中。必须放在段落开头使用，此后整个段落（包括后续行）都会居中对齐。

* {{@@_@@}} — 显式换行。只能用于行尾。
例如，可用于防止两个连续空行被合并为一个段落，如下所示：
[[code]]a
_
_
b[[/code]]

此外，{{@@_@@}} 允许在原本不支持换行的元素中插入换行（例如表格或列表）。
为了正确生效，显式换行符必须与前面的文本或元素之间用空格分隔。

* {{@@~~~@@}}、{{@@~~~<@@}}、{{@@~~~>@@}} — 清除浮动元素。
仅在行首使用时有效。
分别等同于：
{{@@[[div style="clear: both"]][/div]]@@}}
{{@@[[div style="clear: left"]][[/div]]@@}}
{{@@[[div style="clear: right"]][[/div]]@@}}

++ [[# headers]] 标题

{{@@+ 文本@@}}、
{{@@++ 文本@@}}、
{{@@+++ 文本@@}}、
{{@@++++ 文本@@}}、
{{@@+++++ 文本@@}}、
{{@@++++++ 文本@@}} — 分别对应一级至六级标题。

不支持六级以上标题。

该元素只能在新行使用。
内部不支持换行，但可以嵌入块级元素或显式换行，例如：

[[code]]++ 第一行 _
第二行[[/code]]

在标题文本前可以添加符号 {{*}}，例如 {{@@++* 文本@@}}。
此时该标题不会被加入目录。

++ 水平线

{{@@---@@}} — 添加水平分隔线。

该元素只能在新行使用。
前三个 {{-}} 之后的数量没有限制。

++ 引用

以下语法会创建两个嵌套的引用块：

[[code]]
> 第一级
>> 第二级
>> 第二级的第二行
> 回到第一级
[[/code]]

由于 {{@@[[blockquote]]@@}} 具有更好的可读性（和可编辑性），因此不推荐使用此语法。

++ 列表

网站支持三种列表类型：有序列表、无序列表和字典列表。
前两种可以相互嵌套。

示例：无序列表中嵌套有序列表：

[[code]]
* 元素1
* 元素2
 # 元素2.1
 # 元素2.2
* 元素3
[[/code]]

列表的嵌套级别由 {{*}} 或 {{#}} 前的空格或制表符数量决定。

列表项中可以通过 {{_}} 或使用 {{@@[[span]]@@}} 包裹内容来实现多行，例如：

[[code]]
* 元素1 _
下一行
* [[span]]元素2
下一行[[/span]]
[[/code]]

字典列表定义如下：

[[code]]
: 术语1 : 定义
: 术语2 : 定义
: 术语3 : 定义
[[/code]]

在术语或定义中同样可以使用 {{_}} 或 {{@@[[span]]@@}}。

++ 表格

简化（自由）语法的表格如下所示：

[[code]]
||~ 标题 ||~ 标题2 ||
||> 右对齐文本 ||= 居中文本 ||
|||| 横向跨越两列的单元格 ||
[[/code]]

要在单元格中添加多行文本，可以使用 {{_}} 或 {{@@[[span]]@@}}。

该元素无法创建跨越多行（纵向合并）的单元格。在这种情况下，可以使用块级元素 {{@@[[table]]@@}}，它不受此限制。

++ 其他

* {{@@[[# anchor]]@@}} — 创建一个具有指定标识符的元素，从而可以通过链接跳转到该位置（例如：{{@@[#anchor 跳转到锚点]@@}}）。
  该元素在视觉上类似于块级元素，但实际上并不是。
  在元素定义中，{{#}} 后必须保留一个空格。

* {{@@[!-- 注释 --]@@}} — 定义一段在最终页面显示时不会呈现的源代码区域。
  可用于在代码中添加技术性备注。
  [!-- 我就知道你会看到这里。 --]

* [[# literals]]{{@<&#64;&#64;>@文本@<&#64;&#64;>@}} — 阻止 {{@<&#64;&#64;>@}} 包裹的内容被当作标记语法解析。
  始终生成一行文本（字面量）。
  可用于在出于视觉效果使用标记符号时避免歧义（例如，并非链接用途的单个方括号）。
  也可用于破坏自动替换语法（当不希望发生自动替换时），例如：
  {{%%pat@<&#64;&#64;>@h|param%%}} 始终会显示为文本 %%pat@@@@h|param%%，即便指定了参数 {{param}}。
  也可用于像 {{_}} 那样创建空行（但不推荐这样使用）。

* {{@<&#64;&#60;>@&mdash;&copy;@<&#62;&#64;>@}} — 插入指定的 HTML 实体符号（或多个符号）。

* 符号 «、» 和 — 的替换
_
_
由于这些替换不属于自动替换，因此在仅支持纯文本的场景下不会生效（例如块属性或链接名称中）。

 * {{@@ <<@@}} — 左引号：«

 * {{@@ >>@@}} — 右引号：»

 * {{@@ --@@}} — 长破折号：—
   需要注意的是，为了使该元素被解析为破折号而不是删除线语法，其两侧必须有空格。

+ 块级元素

所有块级元素都遵循类似规则构建。

每个块级元素都有名称（例如 {{div}}、{{iftags}}、{{blockquote}} 等）、可选的标识符，以及一个起始标签（例如 {{@@[[div]]@@}}）。
可以包含文本或其他块级元素的块级元素，还必须有与起始标签对应名称的结束标签（例如 {{@@[[/div]]@@}}）。

某些块可以使用修饰符 {{_}}。
该修饰符写在块名称之后，用于阻止在块内自动创建段落（文本会直接放入块中，换行通过 {{<br>}} 实现）。
修饰符只写在起始标签中，因此以下语法是正确的：
[[code]][[div_]]text[[/div]][[/code]]

块的标识符是可选文本，写在块名称之后（但在修饰符 {{_}} 之前），通过 {{:}} 指定。例如：

[[code]]
[[module:lu ListUsers]]
  [[module CSS]]
    body {
      background: url(%%avatar%%);
    }
  [[/module]]
[[/module:lu]]
[[/code]]

块标识符允许在接受文本内容的块（如 {{@@[[code]]@@}}、{{@@[[module]]@@}}、{{@@[[html]]@@}}）内部使用该块的标准结束标签，而不会真正关闭它。
这样可以将多个模块相互嵌套，或在该块内部写出 {{@@[[code]]@@}} 的示例：

[[code:outer]]
[[code:2]]
  [[code:b]]
    示例：使用块 [[cоde]]
  [[/code:b]]
[[/code:2]]
[[/code:outer]]

大多数块可以以某种形式接受属性（要么是 HTML 属性，要么是特定块自定义属性）。
属性写法为 {{参数=值}} 或 {{参数="值"}}。
不同于 HTML，在属性值中使用特殊字符时，不使用 HTML 实体（如 {{&quot;}}），而是通过 {{\}} 转义：
例如 {{@@[[collapsible show="协议 \"忧郁\""]]@@}}。

++ [[# html-attributes]] 标准 HTML 属性

某些元素（例如 {{@@[[a]]@@}}、{{@@[[span]]@@}} 等）是 HTML 的直接接口，
其标记中的属性会直接插入生成的 HTML 页面中。

并非所有 HTML 属性都允许在标记中使用。
允许使用的属性列表如下；更多详情请参阅
https://www.w3schools.com/tags/ref_attributes.asp 的 HTML 文档。
在本网站语境中最常用的属性已用 ##red|红色## 标出。

* {{##red|alt##}}
* {{##red|class##}}
* {{##red|colspan##}}
* {{##red|href##}}
* {{##red|id##}}：该属性会被特殊处理。标识符必须以 {{u-}} 作为前缀。如果未指定此前缀，系统会自动添加。例如，{{id="myid"}} 将会被转换为 {{id="u-myid"}}。
* {{##red|rowspan##}}
* {{##red|style##}}
* {{##red|target##}}
* {{accept}}
* {{align}}
* {{autocapitalize}}
* {{autoplay}}
* {{background}}
* {{bgcolor}}
* {{border}}
* {{buffered}}
* {{checked}}
* {{cite}}
* {{cols}}
* {{contenteditable}}
* {{controls}}
* {{coords}}
* {{datetime}}
* {{decoding}}
* {{default}}
* {{dir}}
* {{dirname}}
* {{disabled}}
* {{download}}
* {{draggable}}
* {{for}}
* {{form}}
* {{headers}}
* {{height}}
* {{hidden}}
* {{high}}
* {{hreflang}}
* {{inputmode}}
* {{ismap}}
* {{itemprop}}
* {{kind}}
* {{label}}
* {{lang}}
* {{list}}
* {{loop}}
* {{low}}
* {{max}}
* {{maxlength}}
* {{min}}
* {{minlength}}
* {{multiple}}
* {{muted}}
* {{name}}
* {{optimum}}
* {{pattern}}
* {{placeholder}}
* {{poster}}
* {{preload}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{role}}
* {{rows}}
* {{scope}}
* {{selected}}
* {{shape}}
* {{size}}
* {{sizes}}
* {{span}}
* {{spellcheck}}
* {{src}}
* {{srclang}}
* {{srcset}}
* {{start}}
* {{step}}
* {{tabindex}}
* {{title}}
* {{translate}}
* {{type}}
* {{usemap}}
* {{value}}
* {{width}}
* {{wrap}}
* {{scrolling}}
* {{frameborder}}

++ [[# booleans]] 布尔属性

文档中标记为可接受布尔值的属性，在实际使用中可以用以下字符串表示：

True：

* {{true}}
* {{t}}
* {{1}}
* {{yes}}

False：

* {{false}}
* {{f}}
* {{0}}
* {{no}}

同样适用于以下内置 HTML 属性：

* {{allowfullscreen}}
* {{allowpaymentrequest}}
* {{async}}
* {{autofocus}}
* {{autoplay}}
* {{checked}}
* {{controls}}
* {{default}}
* {{disabled}}
* {{formnovalidate}}
* {{hidden}}
* {{ismap}}
* {{itemscope}}
* {{loop}}
* {{multiple}}
* {{muted}}
* {{nomodule}}
* {{novalidate}}
* {{open}}
* {{playsinline}}
* {{readonly}}
* {{required}}
* {{reversed}}
* {{selected}}
* {{truespeed}}

++ {{@@[[<]]@@}}、{{@@[[>]]@@}}、{{@@[[=]]@@}}、{{@@[[==]]@@}}：对齐

: 类型 : 全宽
: 段落 : ✅
: 支持属性 : ❌

* {{@@[[<]]@@}} — 将内部文本左对齐。

* {{@@[[>]]@@}} — 将内部文本右对齐。

* {{@@[[=]]@@}} — 将内部文本居中对齐。

* {{@@[[==]]@@}} — 将内部文本两端对齐。

++ {{@@[[a]]@@}}：链接

: 类型 : 行内
: 别名 : {{@@[[anchor]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{*}} : ✅
: 支持 {{_}} : ✅

使用修饰符 {{*}}（例如 {{@@[[*a href="https://google.com"]]Google[[/a]]@@}}）等同于使用 {{target="_blank"}}；该链接会在新窗口（标签页）中打开。
若同时使用该修饰符和 {{target}}，其值将会叠加。

通过该元素创建的链接会经过 [#link-handling 标准过滤]。

++ {{@@[[blockquote]]@@}}：引用块

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[quote]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

在功能上等同于使用 {{>}}，但在标记层面更加“干净”和易读。

++ {{@@[[b]]@@}}：加粗文本

: 类型 : 行内
: 别名 : {{@@[[bold]]@@}}、{{@@[[strong]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[char]]@@}}：HTML 字符

: 类型 : 行内
: 别名 : {{@@[[character]]@@}}
: 支持属性 : ❌

在文本中插入一个 HTML 实体字符。
其工作方式与 {{@<&#64;&#60;>@@<&#62;&#64;>@}} 语法几乎相同。

使用示例：{{@@[[char &mdash;]]@@}}

++ {{@@[[code]]@@}}：代码块

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许忽略 {{@@[[code]]@@}} 与 {{@@[[/code]]@@}} 之间的标记规则。
主要用于展示标记示例而不被立即解析。

该元素也可用于代码高亮。

通过该元素添加到页面的代码，可以通过如下格式的单独文件访问：

{{@@https@@://files.projwikit.unitreaty.org/local@@--@@code/<页面名称>/<页面中的代码块编号，从 1 开始>}}

对于 HTML、JavaScript、XML、CSS 语言，将设置对应的 MIME 类型，
从而可以在 {{<script src>}}、{{<link rel>}} 等需要类型匹配的场景中使用。

仅支持一个属性：

* {{type}} — 需要高亮的编程语言名称。

[[collapsible show="[+] 支持语言完整列表" hide="[-] 支持语言完整列表"]]

||~ 语言名称                ||~ 缩写                ||
|| 1C                      || 1c                     ||
|| ABNF                    || abnf                   ||
|| Access logs             || accesslog              ||
|| Ada                     || ada                    ||
|| Arduino (C++ w/Arduino libs) || arduino, ino           ||
|| ARM assembler           || armasm, arm            ||
|| AVR assembler           || avrasm                 ||
|| ActionScript            || actionscript, as       ||
|| AngelScript             || angelscript, asc       ||
|| Apache                  || apache, apacheconf     ||
|| AppleScript             || applescript, osascript ||
|| Arcade                  || arcade                 ||
|| AsciiDoc                || asciidoc, adoc         ||
|| AspectJ                 || aspectj                ||
|| AutoHotkey              || autohotkey             ||
|| AutoIt                  || autoit                 ||
|| Awk                     || awk, mawk, nawk, gawk  ||
|| Bash                    || bash, sh, zsh          ||
|| Basic                   || basic                  ||
|| BNF                     || bnf                    ||
|| Brainfuck               || brainfuck, bf          ||
|| C#                      || csharp, cs             ||
|| C                       || c, h                   ||
|| C++                     || cpp, hpp, cc, hh, c++, h++, cxx, hxx ||
|| C/AL                    || cal                    ||
|| Cache Object Script    || cos, cls               ||
|| CMake                   || cmake, cmake.in        ||
|| Coq                     || coq                    ||
|| CSP                     || csp                    ||
|| CSS                     || css                    ||
|| Cap’n Proto             || capnproto, capnp       ||
|| Clojure                 || clojure, clj           ||
|| CoffeeScript            || coffeescript, coffee, cson, iced ||
|| Crmsh                   || crmsh, crm, pcmk       ||
|| Crystal                 || crystal, cr            ||
|| D                       || d                      ||
|| Dart                    || dart                   ||
|| Delphi                  || dpr, dfm, pas, pascal  ||
|| Diff                    || diff, patch            ||
|| Django                  || django, jinja          ||
|| DNS Zone file           || dns, zone, bind        ||
|| Dockerfile              || dockerfile, docker     ||
|| DOS                     || dos, bat, cmd          ||
|| dsconfig                || dsconfig               ||
|| DTS (Device Tree)       || dts                    ||
|| Dust                    || dust, dst              ||
|| EBNF                    || ebnf                   ||
|| Elixir                  || elixir                 ||
|| Elm                     || elm                    ||
|| Erlang                  || erlang, erl            ||
|| Excel                   || excel, xls, xlsx       ||
|| F#                      || fsharp, fs             ||
|| FIX                     || fix                    ||
|| Fortran                 || fortran, f90, f95      ||
|| G-Code                  || gcode, nc              ||
|| Gams                    || gams, gms              ||
|| GAUSS                   || gauss, gss             ||
|| Gherkin                 || gherkin                ||
|| Go                      || go, golang             ||
|| Golo                    || golo, gololang         ||
|| Gradle                  || gradle                 ||
|| GraphQL                 || graphql                ||
|| Groovy                  || groovy                 ||
|| HTML, XML               || xml, html, xhtml, rss, atom, xjb, xsd, xsl, plist, svg ||
|| HTTP                    || http, https            ||
|| Haml                    || haml                   ||
|| Handlebars              || handlebars, hbs, html.hbs, html.handlebars        ||
|| Haskell                 || haskell, hs            ||
|| Haxe                    || haxe, hx               ||
|| Hy                      || hy, hylang             ||
|| Ini, TOML               || ini, toml              ||
|| Inform7                 || inform7, i7            ||
|| IRPF90                  || irpf90                 ||
|| JSON                    || json                   ||
|| Java                    || java, jsp              ||
|| JavaScript              || javascript, js, jsx    ||
|| Julia                   || julia, julia-repl      ||
|| Kotlin                  || kotlin, kt             ||
|| LaTeX                   || tex                    ||
|| Leaf                    || leaf                   ||
|| Lasso                   || lasso, ls, lassoscript ||
|| Less                    || less                   ||
|| LDIF                    || ldif                   ||
|| Lisp                    || lisp                   ||
|| LiveCode Server         || livecodeserver         ||
|| LiveScript              || livescript, ls         ||
|| Lua                     || lua                    ||
|| Makefile                || makefile, mk, mak, make ||
|| Markdown                || markdown, md, mkdown, mkd ||
|| Mathematica             || mathematica, mma, wl   ||
|| Matlab                  || matlab                 ||
|| Maxima                  || maxima                 ||
|| Maya Embedded Language  || mel                    ||
|| Mercury                 || mercury                ||
|| Mizar                   || mizar                  ||
|| Mojolicious             || mojolicious            ||
|| Monkey                  || monkey                 ||
|| Moonscript              || moonscript, moon       ||
|| N1QL                    || n1ql                   ||
|| NSIS                    || nsis                   ||
|| Nginx                   || nginx, nginxconf       ||
|| Nim                     || nim, nimrod            ||
|| Nix                     || nix                    ||
|| OCaml                   || ocaml, ml              ||
|| Objective C             || objectivec, mm, objc, obj-c, obj-c++, objective-c++ ||
|| OpenGL Shading Language || glsl                   ||
|| OpenSCAD                || openscad, scad         ||
|| Oracle Rules Language   || ruleslanguage          ||
|| Oxygene                 || oxygene                ||
|| PF                      || pf, pf.conf            ||
|| PHP                     || php                    ||
|| Parser3                 || parser3                ||
|| Perl                    || perl, pl, pm           ||
|| Plaintext               || plaintext, txt, text   ||
|| Pony                    || pony                   ||
|| PostgreSQL & PL/pgSQL   || pgsql, postgres, postgresql ||
|| PowerShell              || powershell, ps, ps1    ||
|| Processing              || processing             ||
|| Prolog                  || prolog                 ||
|| Properties              || properties             ||
|| Protocol Buffers        || protobuf               ||
|| Puppet                  || puppet, pp             ||
|| Python                  || python, py, gyp        ||
|| Python profiler results || profile                ||
|| Python REPL             || python-repl, pycon     ||
|| Q                       || k, kdb                 ||
|| QML                     || qml                    ||
|| R                       || r                      ||
|| ReasonML                || reasonml, re           ||
|| RenderMan RIB           || rib                    ||
|| RenderMan RSL           || rsl                    ||
|| Roboconf                || graph, instances       ||
|| Ruby                    || ruby, rb, gemspec, podspec, thor, irb ||
|| Rust                    || rust, rs               ||
|| SAS                     || SAS, sas               ||
|| SCSS                    || scss                   ||
|| SQL                     || sql                    ||
|| STEP Part 21            || p21, step, stp         ||
|| Scala                   || scala                  ||
|| Scheme                  || scheme                 ||
|| Scilab                  || scilab, sci            ||
|| Shell                   || shell, console         ||
|| Smali                   || smali                  ||
|| Smalltalk               || smalltalk, st          ||
|| SML                     || sml, ml                ||
|| Stan                    || stan, stanfuncs        ||
|| Stata                   || stata                  ||
|| Stylus                  || stylus, styl           ||
|| SubUnit                 || subunit                ||
|| Swift                   || swift                  ||
|| Tcl                     || tcl, tk                ||
|| Test Anything Protocol  || tap                    ||
|| Thrift                  || thrift                 ||
|| TP                      || tp                     ||
|| Twig                    || twig, craftcms         ||
|| TypeScript              || typescript, ts         ||
|| VB.Net                  || vbnet, vb              ||
|| VBScript                || vbscript, vbs          ||
|| VHDL                    || vhdl                   ||
|| Vala                    || vala                   ||
|| Verilog                 || verilog, v             ||
|| Vim Script              || vim                    ||
|| X++                     || axapta, x++            ||
|| x86 Assembly            || x86asm                 ||
|| XL                      || xl, tao                ||
|| XQuery                  || xquery, xpath, xq      ||
|| YAML                    || yml, yaml              ||
|| Zephir                  || zephir, zep            ||

[[/collapsible]]

++ {{@@[[collapsible]]@@}}：可折叠区块

: 类型 : 全宽
: 段落 : ✅
: 支持 HTML 属性 : ❌

创建一个可通过按钮展开或折叠的区块。

除 HTML 属性外，还支持以下属性：

* {{show}} — 区块关闭时按钮显示的文本。

* {{hide}} — 区块打开时按钮显示的文本。

* {{align}} — 按钮文本对齐方式。可选值：
  {{left}}、{{right}}、{{center}}、{{justify}}。

* {{folded}} — 指示区块是否默认展开。[#booleans 布尔值]。

* {{hideLocation}} — 指示展开后关闭按钮的显示位置。
  可选值：{{top}}、{{bottom}}、{{both}}、{{neither}}、{{none}}。
  后两个选项等价。

++ [[# date]] {{@@[[date]]@@}}：日期

: 类型 : 行内
: 支持 HTML 属性 : ❌

插入日期，并根据查看者计算机的时区自动调整显示。

使用特殊属性语法；日期直接写在块名后，而非单独属性：

[[code]]
[[date 2024-02-18T00:00:00Z format="%H:%M:%S %d.%m.%Y"]]
[[/code]]

例如，该日期以 UTC 指定，
但在 UTC+0200 时区的计算机上将显示为：

"02:00:00 18.02.2024"

++ {{@@[[div]]@@}}：全宽通用容器

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

++ [[# footnotes]] {{@@[[footnote]]@@}}：脚注

: 类型 : 行内
: 支持属性 : ❌

在文本中添加编号脚注。
其内容默认显示在页面底部，或显示在 {{@@[[footnoteblock]]@@}} 所在位置（若手动指定）。

++ {{@@[[footnoteblock]]@@}}：脚注块

: 类型 : 全宽
: 支持 HTML 属性 : ❌

显示文章中所有脚注的列表。

支持以下属性：

* {{title}} — 显示在脚注列表上方的标题文本。

* {{hide}} — [#booleans 布尔值]。
  若指定，该脚注块将不显示。
  可用于移除页面自动添加的默认脚注块，使脚注仅以悬浮提示形式显示。

++ {{@@[[form]]@@}}：表单

: 类型 : 全宽
: 段落 : ✅
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

允许填写表单并通过 [#path-params URL 参数] 提交到站内指定页面。

支持所有常规 HTML 表单属性，
但 {{target}} 属性具有特殊含义：它应填写完整的文章标识符，而不是 URL。

使用示例：

[[code]]
[[form target="search"]]
  [[input type="text" name="s"]]
  [[input type="submit"]]
[[/form]]
[[/code]]

点击“提交”后，将跳转到文章 {{search}}，
例如：

{{@@https://projwikit.unitreaty.org/search/s/你的_字符串@@}}

随后可在目标文章中使用 {{@@%%path%%@@}} 功能读取该参数。

++ {{@@[[html]]@@}}: HTML 代码

: 类型 : 全宽
: 段落 : ❌
: 支持属性 : ✅

允许插入任意 HTML 代码，系统会自动将其包裹在 {{@@[[iframe]]@@}} 中。以这种方式创建的框架会自动根据其内容大小进行自适应。

在当前版本的网站中，该元素通过 {{<iframe srcdoc="...">}} 实现。通过这种方式创建的框架没有域名，因此在框架内部的 JS 代码中无法使用 {{window.localStorage}} 和 {{document.cookie}}。

支持以下属性：

* {{external}} — [#booleans 布尔值]（默认 — {{false}}）；启用该参数后，区块将通过媒体域名（https://files.projwikit.unitreaty.org）进行渲染，其方式与 Wikidot 平台相同。此类区块通常加载时间会稍慢几秒，但支持使用 {{window.localStorage}} 和 {{document.cookie}}。
**注意：** 当前页面版本系统在启用 {{external}} 的 HTML 区块中无法正常工作，因此始终会显示你代码的最新版本（包括在预览和查看旧版本页面时）。此外，此类区块不会读取 [#path-params 页面参数]。

++ {{@@[[iframe]]@@}}: 通过链接嵌入外部页面

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

语法：

[[code]]
[[iframe 链接 属性1="值" 属性2="值"]]
[[/code]]

支持所有允许的 HTML 属性。

通过该元素插入的链接会经过 [#link-handling 标准过滤]。

++ [[# if]] {{@@[[#if]]@@}}, {{@@[[if]]@@}}: 根据值是否存在显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

行内示例：

[[code]]
[[#if {$param} | param exists | param does not exist]]
[[/code]]

全宽示例：

[[code]]
[[if {$param}]]
  param exists
[[else]]
  param does not exist
[[/if]]
[[/code]]

只要是非空字符串都被视为“存在”，//除了// {{@@{$变量}@@}} 或 {{@@%%变量%%@@}} 这种格式的字符串。由于未在 {{@@[[include]]@@}} 或模块中传递的参数会保留为文本值，因此可以借此判断参数是否被传入。

++ [[# ifexpr]] {{@@[[#ifexpr]]@@}}, {{@@[[ifexpr]]@@}}: 根据条件表达式显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

块级示例：

[[code]]
[[module CountPages fullname="main"]]
  [[#ifexpr %%count%% > 0 | yes | no]]
[[/module]]
[[/code]]

全宽示例：

[[code]]
[[module CountPages fullname="main"]]
  [[ifexpr %%count%% > 0]]
    main page exists
  [[else]]
    main page does not exist
  [[/ifexpr]]
[[/module]]
[[/code]]

若表达式语法错误（例如变量不存在或参数传递错误），表达式视为 false。

否则，除 {{False}}、{{0}}、{{0.0}} 或 {{""}} 以外的任何值都视为 true。

支持的运算符：{{*}}, {{/}}, {{+}}, {{-}}, {{@@ <<@@}}, {{@@>>@@}}, 以及比较运算 {{==}}, {{!=}}, {{<}}, {{>}}, {{<=}}, {{>=}}。

除直接比较外，大多数运算仅支持数字。字符串必须以 JSON 格式表示（例如 {{"值"}}）。

还支持以下函数：

* {{min(x, y, ...)}} — 返回最小值。
* {{max(x, y, ...)}} — 返回最大值。
* {{abs(x)}} — 返回绝对值。
* {{round(x)}} — 四舍五入为整数。
* {{lower(str)}} — 转为小写。
* {{upper(str)}} — 转为大写。

++ [[# ifcategory]] {{@@[[ifcategory]]@@}}: 根据文章分类显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

示例：

[[code]]
[[ifcategory +theme]]
该文本仅在 theme 分类页面中显示。
[[/ifcategory]]
[[ifcategory -theme]]
该文本在除 theme 外的所有页面显示。
[[/ifcategory]]
[[/code]]

该元素主要用于与 {{[[include]]}} 配合使用，使插入的代码根据所在页面的分类产生不同效果。

若存在多个分类，应以空格分隔。

++ [[# iftags]] {{@@[[iftags]]@@}}: 根据文章标签显示内容

: 类型 : 取决于内容
: 支持属性 : ❌

标签条件写在区块名称后，以空格分隔，格式如下：

* {{+标签}} — 页面必须包含该标签。
* {{-标签}} — 页面不得包含该标签。
* {{标签}} — 页面至少包含列出的其中一个标签。

示例：

[[code]]
[[iftags 对象 故事 +ru]]
该文章属于俄罗斯分部的对象或故事。
[[/iftags]]
[[/code]]

++ {{@@[[image]]@@}}: 图片

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅

插入当前页面附件图片：

[[code]]
[[image img.png alt="my img"]]
[[/code]]

插入其他页面附件图片：

[[code]]
[[image main/ico_arthub.svg alt="image from another page"]]
[[/code]]

也可以插入互联网图片（不推荐，因为可能发生链接失效——上传至站点的文件会随文章长期保留，而网络图片可能随时失效）。

支持对齐修饰符 {{f<}}, {{f>}}, {{<}}, {{>}}, {{=}}。后三种会使图片成为全宽元素。

* {{@@[[f<image]]@@}} — 左浮动图片，文字会环绕。
* {{@@[[f>image]]@@}} — 右浮动图片。
* {{@@[[<image]]@@}} — 左对齐全宽图片。
* {{@@[[>image]]@@}} — 右对齐全宽图片。
* {{@@[[=image]]@@}} — 居中全宽图片。

支持 {{link}} 属性，可将图片自动包裹为链接。该链接会经过 [#link-handling 标准过滤]，效果等同于 {{@@[[a href="..."]][[image ...]][[/a]]@@}}。

++ {{@@[[input]]@@}}: 输入框

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅

可单独使用，也可与表单搭配使用。

++ {{@@[[i]]@@}}: 斜体文本

: 类型 : 行内
: 别名 : {{@@[[italics]]@@}}, {{@@[[em]]@@}}, {{@@[[emphasis]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[lines]]@@}}: 插入空行

: 类型 : 行内
: 支持属性 : ❌

用于插入多行空白。

示例：

[[code]]
[[lines 8]]
[[/code]]

++ {{@@[[ul]]@@}}, {{@@[[ol]]@@}}, {{@@[[li]]@@}}: 列表

: 类型 : 行内 / 全宽
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

* {{@@[[ul]]@@}} — 无序列表（全宽）。
* {{@@[[ol]]@@}} — 有序列表（全宽）。
* {{@@[[li]]@@}} — 列表项（行内）。

示例：

[[code]]
[[ul]]
  [[li]]第一项[[/li]]
  [[li]]第二项[[/li]]
  [[li]]
    [[ol]]
      [[li]]第一条编号[[/li]]
      [[li]]第二条编号[[/li]]
    [[/ol]]
  [[/li]]
  [[li]]第三项[[/li]]
[[/ul]]
[[/code]]

所有列表元素均支持标准 HTML 属性。

++ {{@@[[mark]]@@}}: 高亮文本

: 类型 : 行内
: 别名 : {{@@[[highlight]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# module-block]] {{@@[[module]]@@}}: 插入站点模块

: 类型 : 全宽
: 别名 : {{@@[[module654]]@@}}
: 支持 HTML 属性 : ❌

通用语法：

[[code]]
[[module 模块名称 属性1="值" 属性2="值"]]
模块内容（仅适用于支持内容的模块）
[[/module]]
[[/code]]

模块列表及其属性见“模块”章节。

对于不支持内容的模块（如 {{@@[[module Rate]]@@}}），无需写关闭标签。

++ {{@@[[table]]@@}}, {{@@[[row]]@@}}, {{@@[[hcell]]@@}}, {{@@[[cell]]@@}}: 表格

: 类型 : 全宽
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

类似于标准 HTML 表格。

* {{@@[[table]]@@}} — {{<table>}}
* {{@@[[row]]@@}} — {{<tr>}}
* {{@@[[hcell]]@@}} — {{<th>}}
* {{@@[[cell]]@@}} — {{<td>}}

不支持 {{<thead>}}、{{<tbody>}}、{{<tfoot>}}。

示例：

[[code]]
[[table class="wiki-content-table"]]
  [[row]]
    [[hcell]]标题 1[[/hcell]]
    [[hcell]]标题 2[[/hcell]]
    [[cell rowspan="2"]]跨行单元格[[/cell]]
  [[/row]]
  [[row]]
    [[cell colspan="2"]]跨列单元格[[/cell]]
  [[/row]]
[[/table]]
[[/code]]

++ {{@@[[tabview]]@@}}, {{@@[[tab]]@@}}: 标签页

: 类型 : 全宽
: 段落 : ✅
: 别名 : {{@@[[tabs]]@@}}（对应 {{@@[[tabview]]@@}}）
: 支持 HTML 属性 : ❌

用于在页面中创建可切换的标签页。

示例：

[[code]]
[[tabview]]
  [[tab 标签 1]]
    标签 1 内容
  [[/tab]]
  [[tab 标签 2]]
    标签 2 内容
  [[/tab]]
[[/tabview]]
[[/code]]

标签名称也可使用 {{@@[[tab title="标签名称"]]@@}} 格式。

++ {{@@[[tt]]@@}}: 等宽文本

: 类型 : 行内
: 别名 : {{@@[[mono]]@@}}, {{@@[[monospace]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[p]]@@}}：显式段落

: 类型 : 全宽
: 替代名称 : {{@@[[paragraph]]@@}}
: 段落 : ❌
: 支持 [#html-attributes HTML 属性] : ✅

允许为当前段落指定 HTML 属性，就像为 {{<p>}} 指定属性一样。

示例：

[[code]]
[[p style="color: red"]]
红色段落。
[[/p]]
[[/code]]

++ {{@@[[ruby]]@@}}, {{@@[[rt]]@@}}：汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[rubytext]]@@}}（用于 {{@@[[rt]]@@}}）
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[ruby]]マレニア[[rt]]Malenia[[/rt]][[/ruby]]
[[/code]]

++ {{@@[[rb]]@@}}：简化汉字注音标注

: 类型 : 行内
: 替代名称 : {{@@[[ruby2]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

使用示例：

[[code]]
[[rb マレニア | Malenia]]
[[/code]]

++ {{@@[[size]]@@}}：字体大小

: 类型 : 行内
: 支持属性 : ❌

允许修改字体大小。尺寸可以使用任何 CSS 允许的数值。

~~~

使用示例：

[[code]]
这段文字非常[[size 200%]]大[[/size]]，同时又[[size 6px]]小[[/size]]。
[[/code]]

++ {{@@[[span]]@@}}：行内通用容器

: 类型 : 行内
: 支持 [#html-attributes HTML 属性] : ✅
: 支持 {{_}} : ✅

可以包含任何其他元素，也可用于为文章中的文本添加样式。

++ {{@@[[s]]@@}}：删除线文本

: 类型 : 行内
: 替代名称 : {{@@[[strikethrough]]@@}}, {{@@[[del]]@@}}, {{@@[[deletion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sup]]@@}}：上标文本

: 类型 : 行内
: 替代名称 : {{@@[[super]]@@}}, {{@@[[superscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[sub]]@@}}：下标文本

: 类型 : 行内
: 替代名称 : {{@@[[subscript]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ {{@@[[toc]]@@}}：自动目录

: 类型 : 全宽
: 支持属性 : ❌

在页面中添加一个可展开的标题列表块。

可使用以下前缀改变元素位置：

* {{@@[[f<toc]]@@}} — 左侧浮动块。该块会被周围文本和块级元素环绕。

* {{@@[[f>toc]]@@}} — 右侧浮动块。

++ {{@@[[u]]@@}}：下划线文本

: 类型 : 行内
: 替代名称 : {{@@[[underline]]@@}}, {{@@[[ins]]@@}}, {{@@[[insertion]]@@}}
: 支持 [#html-attributes HTML 属性] : ✅

++ [[# user-block]] {{@@[[user]]@@}}：用户链接

: 类型 : 行内
: 支持属性 : ❌
: 支持 {{*}} : ✅

可以以两种形式使用：

* {{@@[[user 用户名]]@@}} — 输出普通用户链接。

* {{@@[[*user 用户名]]@@}} — 输出用户链接及其头像。

+ [[# link-handling]] 补充：链接处理

出于安全原因，某些链接会被网站屏蔽。

除非某个元素或模块的文档中另有说明，否则所有链接（例如 {{@@[]@@}}、{{@@[[[|]]]@@}}、{{@@[[a href="..."]]@@}}、{{@@[[image ... link="..."]]@@}} 等）都会按照以下规则进行检查。

明确 **禁止** 的绝对链接协议：

* {{data:}}
* {{javascript:}}（除 {{"javascript:;"}} 之外，该形式表示“空链接”）

明确允许的绝对链接协议：

* {{blob:}}
* {{@@chrome-extension://@@}}
* {{@@chrome://@@}}
* {{@@content://@@}}
* {{data:}}
* {{dns:}}
* {{feed:}}
* {{@@file://@@}}
* {{@@ftp://@@}}
* {{@@git://@@}}
* {{@@gopher://@@}}
* {{@@http://@@}}
* {{@@https://@@}}
* {{@@irc6://@@}}
* {{@@irc://@@}}
* {{@@ircs://@@}}
* {{mailto:}}
* {{@@resource://@@}}
* {{@@rtmp://@@}}
* {{@@sftp://@@}}

对于块级元素，下列以指定字符开头的链接同样被视为合法链接：

* {{A-Z}}, {{a-z}}, {{0-9}}, {{.}}（普通相对链接）。
* {{/}}, {{@@//@@}}（基于当前域名和协议的绝对链接）。
* {{#}}（锚点跳转）。
* {{?}}（跳转到当前页面并附带 GET 参数）。
* {{$}}, {{&}}, {{+}}, {{,}}, {{:}}, {{;}}, {{=}}, {{@}}, {{%}}, {{-}}, {{~}}（其他允许用于相对链接的特殊字符）。

对于自由语法元素，会采用更严格的校验规则，并允许更少的特殊字符，以便将链接与语法本身区分开来：

* {{#}} + ({{A-Z}}, {{a-z}}, {{0-9}}, {{_}}, {{-}}, {{%}})（锚点跳转）。
* {{@@//地址@@}}（使用当前协议的绝对链接）。
* {{/地址}}（使用当前域名的绝对链接）。
* 任何不以 {{/}} 开头，但包含它的文本。

+ 补充：模块

模块提供了网站中不属于普通文章的全部功能。这包括交互元素（评分、论坛、标签云）以及扩展功能（向文章添加 CSS 样式、获取其他文章或当前用户的信息等）。

如果模块的运行导致网站功能异常，可以通过添加参数 {{/nomodule/true}} 访问页面（例如：{{@@https://projwikit.unitreaty.org/main/nomodule/true@@}}）。
该参数会完全禁用此页面上的 **所有** 模块。

关于如何在文章中插入模块的详细说明，请参见 [#module-block 章节] {{@@[[module]]@@}}。

此外，需要注意的是，对于支持内部标记的模块（例如 {{ListUsers}}、{{ListPages}} 和 {{CountPages}}），模块内容在技术上并不属于文章内容的一部分。例如，在模块内部定义的任何 [#headers 标题] 或 [#footnotes 脚注]，都只会在该模块范围内显示。

++ 模块 Rate

: 支持参数 : ❌
: 支持内容 : ❌

为文章添加评分组件，效果类似于页面底部“评分”按钮下方的评分模块。

从发布规则角度来看，该模块是必需的，但从技术角度来看并非强制。如果文章中未包含该模块，仍然可以通过“评分”按钮进行投票。

该模块不接受任何参数。

++ 模块 CSS

: 支持参数 : ❌
: 支持内容 : ✅

该模块的内容会原样作为 CSS 样式应用到当前页面。例如：

[[code]]
[[module CSS]]
body {
  background: red;
}
[[/module]]
[[/code]]

可以通过如下地址获取页面上所有 CSS 模块合并后的结果文件：
{{@@https@@://files.projwikit.unitreaty.org/local@@--@@theme/<页面名称>/style.css}}

该方法会处理 {{@@[[if]]@@}}、{{@@[[ifexpr]]@@}} 和 {{@@[[noinclude]]@@}}。
可以通过参数 {{?includeParams=<JSON 格式参数>}} 传递参数。

++ [[# module-listpages]] 模块 ListPages

: 支持参数 : ✅
: 支持内容 : ✅

用于根据指定参数获取文章列表。

对于每一篇找到的文章，模块内容都会被复制并作为标记进行处理，并进行 [#autoreplace 自动替换]，替换以下变量：

* {{%%name%%}} — 文章的自身标识符（例如 {{main}}）。
* {{%%category%%}} — 文章类别（例如 {{sandbox}}）。
* {{%%fullname%%}} — 包含类别的完整标识符（例如 {{sandbox:main}}）。
* {{%%title%%}} — 文章标题。
* {{%%title_linked%%}} — 包含标题的 [#links 内部链接]。
* {{%%link%%}} — 从当前域名开始的绝对链接（以 {{/}} 开头）。
* {{%%content%%}} — 文章内容（标记格式）。
* {{%%rating%%}} — 文章评分。
* {{%%rating_votes%%}} — 文章投票数。
* {{%%current_user_voted%%}} — {{True}}/{{False}}，表示当前用户是否投票。
* {{%%popularity%%}} — 文章人气值：正向投票比例（在点赞系统下）或高于 3.0 的评分比例（在星级系统下）。
* {{%%revisions%%}} — 文章修订次数。
* {{%%index%%}} — 在搜索结果中的序号。
* {{%%total%%}} — 符合条件的文章总数。
* {{%%created_by%%}} — 创建文章的用户名。
* {{%%created_by_linked%%}} — [#user-block 创建者的用户名、头像及个人主页链接]。
* {{%%updated_by%%}} — 最后编辑者用户名。
* {{%%updated_by_linked%%}} — [#user-block 最后编辑者的用户名、头像及个人主页链接]。
* {{%%tags%%}} — 以逗号分隔的标签列表。
* {{%%tags_linked%%}} — 以逗号分隔的标签链接列表，格式为 {{/system:page-tags/tag/标签名}}。
* {{%%created_at%%}} — [#date 创建日期]。
* {{%%updated_at%%}} — [#date 最后修改日期]。

ListPages 模块参数列表：

* {{range}} — 只能使用 {{range="."}} 格式。
用于优化当前页面变量获取。
使用该参数时，其它参数将被忽略。

* {{fullname}} — 限制为指定完整标识符的单一文章。
可使用 {{.}} 表示当前文章。
使用该参数时，其它参数将被忽略。

* {{pagetype}} — 按文章类型筛选。
类型可为 {{hidden}}（标识符以 {{_}} 开头）或 {{normal}}（其它文章）。
默认值为 {{normal}}。

* {{name}} — 按文章自身标识符筛选（不含类别）。
例如 {{name="main"}} 可能匹配 {{wl:main}} 或 {{sandbox:main}}。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{.}} — 当前文章（使用后忽略其它参数）。
 * {{=}} — 与当前文章相同标识符。
 * {{文本%}} 或 {{文本*}} — 指定前缀。
 * 具体标识符 — 精确匹配。

* {{tags}} — 按标签筛选。
可接受值：

 * {{*}}（默认）— 不限制。
 * {{-}} — 无标签文章。
 * {{=}} — 至少包含当前文章标签（允许额外标签）。
 * {{==}} — 标签完全一致。
 * [#iftags iftags 表达式]。

* {{category}} — 按类别筛选。
可接受值：

 * {{*}} — 不限制。
 * {{.}}（默认）— 当前类别。
 * [#ifcategory ifcategory 表达式]。

* {{parent}} — 按父页面筛选。
可接受值：

 * {{-}} — 无父页面。
 * {{=}} — 与当前页面相同父页面。
 * {{-=}} — 与当前页面不同父页面。
 * {{.}} — 父页面为当前页面。
 * 完整标识符 — 指定父页面。

* {{created_by}} — 按作者筛选。
可接受值：

 * {{.}} — 当前用户。
 * 用户名 — 指定作者。

* {{created_at}} — 按创建日期筛选。
格式 {{年-月-日}}（月日可省略）。
支持比较运算：{{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{rating}} — 按评分筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{votes}} — 按投票数筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{popularity}} — 按人气值筛选。
##red|注意：添加该参数会显著降低查询速度。##
支持 {{=}}, {{<>}}, {{>=}}, {{>}}, {{<=}}, {{<}}。

* {{order}} — 排序。
默认升序。降序需添加 {{desc}}，例如 {{order="created_at desc"}}。
支持排序字段：

 * {{created_at}}
 * {{created_by}}
 * {{updated_at}}
 * {{name}}
 * {{fullname}}
 * {{title}}
 * {{rating}} ##red|(注意：会显著降低查询速度)##
 * {{votes}} ##red|(注意：会显著降低查询速度)##
 * {{popularity}} ##red|(注意：会显著降低查询速度)##
 * {{random}} — 随机排序。

* {{offset}} — 相对于第一篇找到的文章的偏移量，从该位置开始渲染模块；例如 {{offset="1"}} 表示第一篇找到的文章将不会被显示。

* {{limit}} — 模块中最多渲染的文章数量。

* {{perpage}} — 模块单页最多显示的文章数量。
若超过该值，模块中将出现分页列表，可在页面之间切换。
注意：当参数 {{wrapper}} 被设置为负值时，文章数量仍会受到限制，但分页列表 **不会** 出现。

* {{p}} — 当文章数量超过一页时，模块的初始页码。

此外，还可以使用以下参数控制文章信息的输出方式：

* {{prependLine}} — 此参数中的标记将在文章列表渲染之前输出；通常用于表格标题行。
该参数中不支持变量。

* {{appendLine}} — 此参数中的标记将在文章列表渲染完成后输出。
该参数中不支持变量。

* {{separate}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，每篇文章的标记（以及 {{appendLine}} 和 {{prependLine}}）都会作为完全独立的元素处理。
这主要影响 {{@@[[iftags]]@@}} 与 {{@@[[ifcategory]]@@}} 的行为、通过 {{@@[[image]]@@}} 使用这些文章中的图片，以及是否可以将标记拼接为完整代码（例如用于表格）。
因此，若使用 ListPages 渲染表格，必须设置 {{separate="false"}}。

* {{wrapper}} — [#booleans 布尔值]（默认值为 {{true}}）；启用后，模块内容会被包裹在带有 {{list-pages-box}} 类名的 {{<div>}} 元素中。
同时启用 ListPages 的动态（AJAX）功能，例如分页切换。
不建议关闭此参数。

* {{reverse}} — [#booleans 布尔值]（默认值为 {{false}}）；启用后，文章列表顺序将被反转。
该操作将在通过 {{order}} 参数完成普通排序之后执行。

对于上述任意属性，都可以通过 {{@URL@@|@@默认值}} 从页面地址中的变量获取值。
该结构会将模块属性设置为与属性同名的 URL 变量值，若未指定则使用 {{默认值}}。例如，若目标文章包含 {{@@[[module ListPages category="@URL@@|@@sandbox"]]@@}}，并通过 {{/category/fragment}} 访问页面，则模块将输出 {{fragment}} 分类下的页面；若未传递变量，则输出 {{sandbox}} 分类下的页面。

++ 模块 CountPages

: 支持参数 : ✅
: 支持内容 : ✅

支持与 [#module-listpages 模块 ListPages] 相同的参数，但不会输出任何文章信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%total%%}}, {{%%count%%}} — 符合条件的文章数量。

++ 模块 ListUsers

: 支持参数 : ✅
: 支持内容 : ✅

该模块可用于获取当前用户的信息。

可包含标记内容，并通过 [#autoreplace 自动替换] 使用以下变量：

* {{%%number%%}} — 用户 ID。
* {{%%title%%}}, {{%%name%%}} — 用户名。
* {{%%avatar%%}} — 用户头像链接。

默认情况下，若当前用户未登录，模块将不会显示。

可使用布尔参数 {{always}} 防止该行为。
当其为正值时，模块内容仍会显示，但除 {{%%avatar%%}}（默认头像链接）外，其它变量将不可用。

因此，可以使用以下方式检测用户是否已登录：

[[code]]
[[module ListUsers always="yes"]]
  [[if %%number%%]]
    用户已登录
  [[else]]
    用户未登录
  [[/if]]
[[/module]]
[[/code]]

++ 模块 Redirect

: 支持参数 : ✅
: 支持内容 : ❌

将当前页面重定向至另一页面。支持参数：

* {{destination}} — 目标地址。
不会进行标准过滤，仅禁止以 {{data:}} 或 {{javascript:}} 开头的链接。

* {{noredirect}} — 当为正的 [#booleans 布尔值] 时，阻止模块在当前页面生效。
通常在编辑包含该模块的页面时使用（{{/noredirect/true}}）。

示例：

[[code]]
[[module Redirect destination="/scp-1730"]]
[[/code]]

++ 模块 InterWiki

: 支持参数 : ✅
: 支持内容 : ✅

系统模块。用于在侧边栏显示当前页面的多语言翻译。

通过 API [https://crom.avn.sh/ Crom] 实现，由英文 SCP 社区创建并维护。

对于每个找到的翻译，模块内容会被复制并作为标记处理，并进行 [#autoreplace 自动替换]，可使用以下变量：

* {{%%url%%}} — 翻译页面地址。
* {{%%language%%}} — 翻译语言名称或对应维基名称（若同一语言有多个维基），语言由 {{language}} 参数指定。
* {{%%language_native%%}} — 翻译语言在其自身语言中的名称。
* {{%%language_code%%}} — 翻译语言代码（例如 {{en}}, {{ru}}）。

InterWiki 模块参数：

* {{article}} — 要获取翻译列表的文章名称。

* {{prependLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染前输出。
不支持变量。

* {{appendLine}} — 若翻译数量超过一个，该参数中的标记将在翻译列表渲染后输出。
不支持变量。

* {{language}} — {{%%language%%}} 所使用的显示语言。

* {{order}} — 排序方向与变量。可使用 {{url}}, {{language}}, {{language_native}}, {{language_code}}。
可添加 {{desc}} 表示降序（例如 {{order="language_native desc"}}）。

* {{omitlanguage}} — 在获取翻译列表时忽略的语言（通常为当前维基语言）。

* {{empty}} — 若未找到任何翻译时显示的标记。

* {{loading}} — 页面加载完成后立即显示，在获取翻译数据之前显示的标记。

++ 模块 TagCloud 与 PagesByTag

系统模块，用于支持页面 <<[[[system:page-tags|标签云]]]>> 的功能。

++ 模块 SiteChanges

系统模块，用于支持页面 <<[[[system:recent-changes|最近更改]]]>> 的功能。

++ 模块 ForumStart、ForumCategory、ForumThread、ForumNewThread、ForumNewPost

系统模块，用于支持论坛功能。

* ForumStart 模块必须位于系统页面 {{forum:start}}，用于显示分区列表。
* ForumCategory 模块必须位于系统页面 {{forum:category}}，用于显示指定分区的主题列表。
* ForumThread 模块必须位于系统页面 {{forum:thread}}，用于显示指定主题中的帖子。
* ForumNewThread 模块必须位于系统页面 {{forum:new-thread}}，用于在指定分区创建新主题。
* ForumNewPost 不直接用于页面，而通过主题页面的模块 API 使用。

++ 模块 RecentPosts

系统模块，用于支持页面 <<[[[forum:recent-posts|论坛最新帖子]]]>> 的功能。

[[/div]]
', NULL, '2026-08-26 16:12:44.696613+00', 118, NULL);
INSERT INTO public.web_articleversion VALUES (27, '[[module search]]', NULL, '2026-08-26 16:12:44.753787+00', 119, NULL);
INSERT INTO public.web_articleversion VALUES (28, '[[module RecentPosts]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', NULL, '2026-08-26 16:12:44.769568+00', 120, NULL);
INSERT INTO public.web_articleversion VALUES (29, '[[module ForumNewThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', NULL, '2026-08-26 16:12:44.782571+00', 121, NULL);
INSERT INTO public.web_articleversion VALUES (30, '[[module ForumCategory]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', NULL, '2026-08-26 16:12:44.794816+00', 122, NULL);
INSERT INTO public.web_articleversion VALUES (31, '[[module ForumThread]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]
', NULL, '2026-08-26 16:12:44.809024+00', 123, NULL);
INSERT INTO public.web_articleversion VALUES (32, '[[div class="new-post"]]
[[[forum:recent-posts|论坛新帖]]]
[[/div]]

[[module ForumStart]]

[!-- 如果您希望论坛正常工作，请不要更改此页面 --]', NULL, '2026-08-26 16:12:44.821858+00', 124, NULL);
INSERT INTO public.web_articleversion VALUES (33, '* [# 这里]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 是]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 一个]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 示例]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
 * [[[main|鱼！]]]
* [# 顶部栏]
 * [https://github.com/WikitTeam/ProjectWikit GitHub页面]
 * [[[/forum/start|论坛]]]
 * [[[/forum:recent-posts|最新帖子]]]
 * [[[/wiki-syntax-guide|维基语法指南]]]
', NULL, '2026-08-26 16:12:44.837248+00', 125, NULL);
INSERT INTO public.web_articleversion VALUES (34, '[[div class="top-bar"]]
[[include nav:top-impl]]
[[/div]]

[[div class="mobile-top-bar"]]
[[div class="open-menu"]]
[#side-bar ≡]
[[/div]]
[[include nav:top-impl]]
[[/div]]', NULL, '2026-08-26 16:12:44.848137+00', 1, NULL);
INSERT INTO public.web_articleversion VALUES (35, '++ 存在的页面：

[[module listpages category="*" separate="False" prependLine="|| **标题** || **名称** ||"]]
|| %%title_linked%% || %%fullname%% ||
[[/module]]', NULL, '2026-08-26 16:12:44.861191+00', 2, NULL);
INSERT INTO public.web_articleversion VALUES (36, '[[*user Probe WD]]', NULL, '2026-08-26 16:49:09.127295+00', 126, NULL);
INSERT INTO public.web_articleversion VALUES (37, '[[module ListPages category="probe" order="name" perPage="3" separate="yes"]]
%%index%%/%%total%% [[[%%fullname%%|%%title%%]]] %%rating%%
[[/module]]', NULL, '2026-08-27 10:35:07.177917+00', 127, NULL);
INSERT INTO public.web_articleversion VALUES (38, '[[module ListPages category="probe" order="name" separate="no" prependline="||~ page||" appendline="end"]]
||%%name%%||
[[/module]]', NULL, '2026-08-27 10:35:07.258253+00', 128, NULL);
INSERT INTO public.web_articleversion VALUES (39, '[[module ListPages category="probe" order="name" wrapper="no" limit="2"]]
%%name%%
[[/module]]', NULL, '2026-08-27 10:35:07.303821+00', 129, NULL);
INSERT INTO public.web_articleversion VALUES (40, '[[module ListPages category="probe" order="name" perPage="2"]]
[[head]]
top
[[/head]]
[[body]]
%%name%%
[[/body]]
[[foot]]
bottom
[[/foot]]
[[/module]]', NULL, '2026-08-27 10:35:07.335602+00', 130, NULL);
INSERT INTO public.web_articleversion VALUES (41, '[[module ListPages category="*" tags="+lang:en -zeta" order="fullname"]]
%%fullname%%
[[/module]]', NULL, '2026-08-27 10:35:07.368899+00', 131, NULL);
INSERT INTO public.web_articleversion VALUES (42, '[[module ListPages category="probe" name="no-such-name-at-all"]]
%%name%%
[[/module]]', NULL, '2026-08-27 10:35:07.397895+00', 132, NULL);
INSERT INTO public.web_articleversion VALUES (43, '[[module ListPages category="probe" order="name" name="@url|probe*"]]
%%name%%
[[/module]]', NULL, '2026-08-27 10:35:07.430044+00', 133, NULL);
INSERT INTO public.web_articleversion VALUES (44, '[[module ListPages category="*" order="votes desc" perPage="5" rating=">-10"]]
%%fullname%% %%rating%% %%rating_votes%% %%popularity%%
[[/module]]', NULL, '2026-08-27 10:35:07.482231+00', 134, NULL);
INSERT INTO public.web_articleversion VALUES (49, '[[module Rate]]', NULL, '2026-08-28 13:17:07.540564+00', 139, NULL);
INSERT INTO public.web_articleversion VALUES (50, '[[module CSS]]
#page-content { color : red ; }
@media (max-width: 767px) { #main { padding : 0 ; } }
[[/module]]
styled body', NULL, '2026-08-30 13:49:22.291121+00', 196, NULL);
INSERT INTO public.web_articleversion VALUES (51, '[[module CSS head="true"]]
.a { color: #ff0000 }
[[/module]]
head styled body', NULL, '2026-08-30 13:49:22.365165+00', 197, NULL);
INSERT INTO public.web_articleversion VALUES (52, '[[module SiteChanges]]', NULL, '2026-08-31 04:00:17.82285+00', 198, NULL);
INSERT INTO public.web_articleversion VALUES (45, 'unrated source
[[[probe:no-such-one|first]]] [[[probe:no-such-two]]] [[[wanted:alpha|alpha]]]', NULL, '2026-08-28 13:10:35.121649+00', 135, NULL);
INSERT INTO public.web_articleversion VALUES (47, 'quarter source
[[[wanted:beta]]] [[[probe:no-such-one]]]', NULL, '2026-08-28 13:10:35.367772+00', 137, NULL);
INSERT INTO public.web_articleversion VALUES (48, 'unratable source
[[[wanted:gamma]]] [[[probe:full|a page that exists]]]', NULL, '2026-08-28 13:10:35.399908+00', 138, NULL);
INSERT INTO public.web_articleversion VALUES (67, '+ pwikit 新功能演示

这一页把 2026-08-31 加进来的四个模块和几项站点设置放在一起看。

++ 一、@@[[module time]]@@ —— 当前时间

现在是 **[[module time]]%%currentyear%% 年 %%currentmonth%% 月 %%currentday%% 日 %%currenthour%%:%%currentminute%%[[/module]]**

年、月、日、时、分各自是一个 {{odate}} 元素，**由读者自己的浏览器按本地时区渲染** —— 换个时区打开这一页，
上面的数字会跟着变；没开 JS 的读者看到的是服务端那一侧的值。可用的变量：

* @@%%currentyear%%@@ 年（四位）
* @@%%currentmonth%%@@ 月（补零，{{08}} 不是 {{8}}）
* @@%%currentday%%@@ 日 ｜ @@%%currenthour%%@@ 时 ｜ @@%%currentminute%%@@ 分 ｜ @@%%currentsecond%%@@ 秒

少一个 r 的 @@%%curentyear%%@@ 也认。**这个模块是有正文的，要写 @@[[/module]]@@。**

> **已知代价**：模块在 ftml 里是块级的，所以它自成一段，句子中间嵌不进去。
> 页脚是唯一的例外 —— 那里由 Go 自己渲染并把段落剥掉。

++ 二、@@[[module members]]@@ —— 全站成员

按用户编号排序，默认每页 100 人（这里用 @@perpage="10"@@），@@%%index%%@@ 是**跨页连续**的自动编号。

||~ # ||~ 用户 ||~ 编号 ||~ 注册时间 ||
[[module members perpage="10"]]
|| %%index%% || %%members%% || %%number%% || %%time%% ||
[[/module]]

++ 三、只列某一个身分组

加 @@role="editor"@@ 就只列那个组的人，收标识符或后台里的编号。

||~ # ||~ 「成员」组 ||~ 注册时间 ||
[[module members role="editor" perpage="10"]]
|| %%index%% || %%members%% || %%time%% ||
[[/module]]

++ 四、@@[[module applicationform]]@@ —— 工单

**不需要 @@[[/module]]@@，一行写完。** 不带参数是普通工单，进后台的「用户工单」，
标题栏交给提交人自己填。**没登录时只出一句提示，不出表单。**

[[module applicationform]]

++ 五、@@title="..."@@ —— 标题由页面写死

带 @@type="membershipapply"@@ 是入组申请，进后台的「申请书」，管理员点详情可以直接决定发哪个身分组，
改成「已通过」当场就发下去。这里再加一个 @@title="入组申请"@@，**标题栏就不出现了，提交上去的标题固定是这一句**。

[[module applicationform type="membershipapply" title="入组申请"]]

++ 六、@@title="no"@@ —— 干脆不要标题

同一个参数，写成 @@yes@@／@@no@@ 就是开关，写成别的就是标题本身。这一个连标题都不提交。

[[module applicationform title="no"]]

++ 七、@@[[module membershipbypassword]]@@ —— 密码入组

也**不需要 @@[[/module]]@@**。后台**默认不启用**，关着的时候这一整块一个字都不输出 —— 不去暴露它存在。
这个演示站已经打开了它，口令是 @@letmein@@，输对了发「成员」组。左边那句默认是「输入密码」：

[[module membershipbypassword]]

加 @@label="..."@@ 就换成自己的说法：

[[module membershipbypassword label="持有邀请口令？填在这里"]]

++ 八、页脚

往下看页脚那一行 —— 它是后台里可以自己写的 wikitext，**并且只放行 @@[[module time]]@@ 这一个模块**。
在那里写 @@[[module listpages]]@@ 只会得到一个「模块不存在」的报错块。

++ 九、后台在哪

* **站点设置** —— [/-/admin/web/site/1/change/ /-/admin/web/site/1/change/]：站点图标、登录页图标、页脚、注册提示、三种身分组、密码入组
* **用户工单** —— [/-/admin/web/supportticket/ /-/admin/web/supportticket/]
* **申请书** —— [/-/admin/web/membershipapplication/ /-/admin/web/membershipapplication/]
* **角色与权限** —— [/-/admin/web/role/ /-/admin/web/role/]，新增的那一项叫「访问权限表」
', NULL, '2026-08-31 11:20:09.763556+00', 205, NULL);

INSERT INTO public.web_category VALUES (1, 'probestars', true);
INSERT INTO public.web_category VALUES (96, 'probeoff', true);

INSERT INTO public.web_externallink VALUES (25, 'probestars:quarter', 'wanted:beta', 'link');
INSERT INTO public.web_externallink VALUES (26, 'probestars:quarter', 'probe:no-such-one', 'link');
INSERT INTO public.web_externallink VALUES (27, 'scp-173', 'component:box', 'include');
INSERT INTO public.web_externallink VALUES (28, 'scp-173', 'main', 'link');
INSERT INTO public.web_externallink VALUES (29, 'scp-173', 'another-missing', 'link');
INSERT INTO public.web_externallink VALUES (30, 'probe:host', 'probe:included', 'include');
INSERT INTO public.web_externallink VALUES (31, 'probe-a', 'component:probe-var', 'include');
INSERT INTO public.web_externallink VALUES (32, 'probe-b', 'component:probe-var', 'include');
INSERT INTO public.web_externallink VALUES (33, 'forum:start', 'forum:recent-posts', 'link');
INSERT INTO public.web_externallink VALUES (34, 'probeoff:unratable', 'wanted:gamma', 'link');
INSERT INTO public.web_externallink VALUES (35, 'probeoff:unratable', 'probe:full', 'link');
INSERT INTO public.web_externallink VALUES (36, 'nav:top-impl', 'main', 'link');
INSERT INTO public.web_externallink VALUES (37, 'nav:top', 'nav:top-impl', 'include');
INSERT INTO public.web_externallink VALUES (38, 'wiki-syntax-guide', 'component:_template', 'link');
INSERT INTO public.web_externallink VALUES (39, 'wiki-syntax-guide', '_template', 'link');
INSERT INTO public.web_externallink VALUES (40, 'wiki-syntax-guide', 'system:page-tags', 'link');
INSERT INTO public.web_externallink VALUES (41, 'wiki-syntax-guide', 'system:recent-changes', 'link');
INSERT INTO public.web_externallink VALUES (42, 'wiki-syntax-guide', 'forum:recent-posts', 'link');
INSERT INTO public.web_externallink VALUES (43, 'probestars:unrated', 'probe:no-such-one', 'link');
INSERT INTO public.web_externallink VALUES (44, 'probestars:unrated', 'probe:no-such-two', 'link');
INSERT INTO public.web_externallink VALUES (45, 'probestars:unrated', 'wanted:alpha', 'link');

INSERT INTO public.web_file VALUES (1, 'probe attach.pdf', '11111111-2222-3333-4444-555555555555', 'application/pdf', 300, '2026-08-23 04:41:28.859818+00', NULL, 4, NULL, NULL);

INSERT INTO public.web_forumsection VALUES (71, 'Probe Open', 'an open section', 0, false, false);
INSERT INTO public.web_forumsection VALUES (72, 'Probe Hidden', 'hidden from the listing', 1, true, false);
INSERT INTO public.web_forumsection VALUES (73, 'Probe Staff', 'only staff may browse', 2, false, true);

INSERT INTO public.web_forumcategory VALUES (55, 'Probe Chat', 'ordinary threads', 0, false, 71);
INSERT INTO public.web_forumcategory VALUES (56, 'Probe Comments', 'article comments', 1, true, 71);
INSERT INTO public.web_forumcategory VALUES (57, 'Probe Quiet', 'nobody has posted here', 2, false, 71);
INSERT INTO public.web_forumcategory VALUES (58, 'Probe Hidden Chat', 'inside the hidden section', 0, false, 72);
INSERT INTO public.web_forumcategory VALUES (59, 'Probe Staff Chat', 'inside the staff section', 0, false, 73);
INSERT INTO public.web_forumcategory VALUES (60, 'Probe Busy', 'enough threads for a second page', 3, false, 71);
INSERT INTO public.web_forumcategory VALUES (61, 'Probe Talk', 'threads that carry replies', 4, false, 71);

INSERT INTO public.web_forumthread VALUES (1, '', '', 4, NULL, NULL, false, '2026-08-20 07:28:44.767734+00', '2026-08-20 07:28:44.767767+00', false);
INSERT INTO public.web_forumthread VALUES (2, '', '', 5, NULL, NULL, false, '2026-08-20 07:28:47.858291+00', '2026-08-20 07:28:47.858315+00', false);
INSERT INTO public.web_forumthread VALUES (3, '', '', 3, NULL, NULL, false, '2026-08-20 07:28:48.304755+00', '2026-08-20 07:28:48.30477+00', false);
INSERT INTO public.web_forumthread VALUES (4, '', '', 1, NULL, NULL, false, '2026-08-20 07:28:48.762059+00', '2026-08-20 07:28:48.762073+00', false);
INSERT INTO public.web_forumthread VALUES (5, '', '', 2, NULL, NULL, false, '2026-08-20 07:28:49.310702+00', '2026-08-20 07:28:49.310717+00', false);
INSERT INTO public.web_forumthread VALUES (9, '', '', 9, NULL, NULL, false, '2026-08-23 08:47:23.179562+00', '2026-08-23 08:47:23.179592+00', false);
INSERT INTO public.web_forumthread VALUES (10, '', '', 10, NULL, NULL, false, '2026-08-23 08:47:24.460298+00', '2026-08-23 08:47:24.460309+00', false);
INSERT INTO public.web_forumthread VALUES (11, '', '', 8, NULL, NULL, false, '2026-08-23 08:47:24.741236+00', '2026-08-23 08:47:24.741268+00', false);
INSERT INTO public.web_forumthread VALUES (12, '', '', 17, NULL, NULL, false, '2026-08-24 10:56:38.756608+00', '2026-08-24 10:56:38.756639+00', false);
INSERT INTO public.web_forumthread VALUES (98, 'Probe Locked Thread', 'Probe Locked Thread description', NULL, 32, 55, false, '2021-03-04 05:07:07+00', '2022-07-08 09:09:11+00', true);
INSERT INTO public.web_forumthread VALUES (14, 'Probe Full', '', 12, NULL, NULL, false, '2026-08-26 13:01:06.560911+00', '2026-08-26 13:01:06.560938+00', false);
INSERT INTO public.web_forumthread VALUES (15, '', '', 13, NULL, NULL, false, '2026-08-26 13:33:17.502922+00', '2026-08-26 13:33:17.503124+00', false);
INSERT INTO public.web_forumthread VALUES (16, '', '', 11, NULL, NULL, false, '2026-08-26 13:33:21.091869+00', '2026-08-26 13:33:21.091954+00', false);
INSERT INTO public.web_forumthread VALUES (17, '', '', 14, NULL, NULL, false, '2026-08-26 13:34:39.712839+00', '2026-08-26 13:34:39.713147+00', false);
INSERT INTO public.web_forumthread VALUES (18, '', '', 15, NULL, NULL, false, '2026-08-26 13:34:41.249561+00', '2026-08-26 13:34:41.249596+00', false);
INSERT INTO public.web_forumthread VALUES (19, '', '', 16, NULL, NULL, false, '2026-08-26 13:34:47.861358+00', '2026-08-26 13:34:47.861495+00', false);
INSERT INTO public.web_forumthread VALUES (20, '', '', 113, NULL, NULL, false, '2026-08-26 14:01:00.65422+00', '2026-08-26 14:01:00.654351+00', false);
INSERT INTO public.web_forumthread VALUES (21, '', '', 114, NULL, NULL, false, '2026-08-26 14:01:04.103173+00', '2026-08-26 14:01:04.103276+00', false);
INSERT INTO public.web_forumthread VALUES (22, '', '', 115, NULL, NULL, false, '2026-08-26 14:01:07.352869+00', '2026-08-26 14:01:07.352922+00', false);
INSERT INTO public.web_forumthread VALUES (23, '', '', 116, NULL, NULL, false, '2026-08-26 14:01:10.593731+00', '2026-08-26 14:01:10.593752+00', false);
INSERT INTO public.web_forumthread VALUES (24, '', '', 117, NULL, NULL, false, '2026-08-26 14:01:14.590565+00', '2026-08-26 14:01:14.590618+00', false);
INSERT INTO public.web_forumthread VALUES (25, '', '', 112, NULL, NULL, false, '2026-08-26 14:01:17.193583+00', '2026-08-26 14:01:17.193661+00', false);
INSERT INTO public.web_forumthread VALUES (26, '', '', 124, NULL, NULL, false, '2026-08-26 16:12:57.817666+00', '2026-08-26 16:12:57.817881+00', false);
INSERT INTO public.web_forumthread VALUES (27, '', '', 118, NULL, NULL, false, '2026-08-26 16:13:06.365908+00', '2026-08-26 16:13:06.365928+00', false);
INSERT INTO public.web_forumthread VALUES (28, '', '', 119, NULL, NULL, false, '2026-08-26 16:13:08.482061+00', '2026-08-26 16:13:08.48225+00', false);
INSERT INTO public.web_forumthread VALUES (29, '', '', 126, NULL, NULL, false, '2026-08-26 16:49:16.945257+00', '2026-08-26 16:49:16.945373+00', false);
INSERT INTO public.web_forumthread VALUES (30, '', '', 125, NULL, NULL, false, '2026-08-26 16:54:12.215069+00', '2026-08-26 16:54:12.215102+00', false);
INSERT INTO public.web_forumthread VALUES (31, '', '', 123, NULL, NULL, false, '2026-08-26 16:54:14.777505+00', '2026-08-26 16:54:14.777592+00', false);
INSERT INTO public.web_forumthread VALUES (32, '', '', 122, NULL, NULL, false, '2026-08-26 16:54:15.699462+00', '2026-08-26 16:54:15.69955+00', false);
INSERT INTO public.web_forumthread VALUES (33, '', '', 121, NULL, NULL, false, '2026-08-26 16:54:17.673113+00', '2026-08-26 16:54:17.673126+00', false);
INSERT INTO public.web_forumthread VALUES (34, '', '', 120, NULL, NULL, false, '2026-08-26 16:54:18.474567+00', '2026-08-26 16:54:18.474613+00', false);
INSERT INTO public.web_forumthread VALUES (35, '', '', 127, NULL, NULL, false, '2026-08-27 10:51:04.277772+00', '2026-08-27 10:51:04.277851+00', false);
INSERT INTO public.web_forumthread VALUES (36, '', '', 128, NULL, NULL, false, '2026-08-27 10:52:01.569373+00', '2026-08-27 10:52:01.569493+00', false);
INSERT INTO public.web_forumthread VALUES (37, '', '', 129, NULL, NULL, false, '2026-08-27 10:52:03.356+00', '2026-08-27 10:52:03.356153+00', false);
INSERT INTO public.web_forumthread VALUES (38, '', '', 130, NULL, NULL, false, '2026-08-27 10:52:05.059031+00', '2026-08-27 10:52:05.059078+00', false);
INSERT INTO public.web_forumthread VALUES (39, '', '', 131, NULL, NULL, false, '2026-08-27 10:52:06.407958+00', '2026-08-27 10:52:06.408088+00', false);
INSERT INTO public.web_forumthread VALUES (40, '', '', 132, NULL, NULL, false, '2026-08-27 10:52:08.207255+00', '2026-08-27 10:52:08.207399+00', false);
INSERT INTO public.web_forumthread VALUES (41, '', '', 133, NULL, NULL, false, '2026-08-27 10:52:09.865071+00', '2026-08-27 10:52:09.865093+00', false);
INSERT INTO public.web_forumthread VALUES (42, '', '', 134, NULL, NULL, false, '2026-08-27 10:52:11.216288+00', '2026-08-27 10:52:11.216405+00', false);
INSERT INTO public.web_forumthread VALUES (43, '', '', 139, NULL, NULL, false, '2026-08-28 13:17:39.423178+00', '2026-08-28 13:17:39.423394+00', false);
INSERT INTO public.web_forumthread VALUES (106, 'Probe Busy 06', 'Probe Busy 06 description', NULL, 30, 60, false, '2021-03-04 05:15:07+00', '2022-07-08 09:01:11+00', false);
INSERT INTO public.web_forumthread VALUES (112, 'Probe Busy 12', 'Probe Busy 12 description', NULL, 30, 60, false, '2021-03-04 05:21:07+00', '2022-07-08 08:55:11+00', false);
INSERT INTO public.web_forumthread VALUES (115, 'Probe Busy 15', 'Probe Busy 15 description', NULL, 30, 60, false, '2021-03-04 05:24:07+00', '2022-07-08 08:52:11+00', false);
INSERT INTO public.web_forumthread VALUES (99, 'Probe Pinned Thread', 'Probe Pinned Thread description', NULL, 32, 55, true, '2021-03-04 05:08:07+00', '2022-07-08 09:08:11+00', false);
INSERT INTO public.web_forumthread VALUES (117, 'Probe Busy 17', 'Probe Busy 17 description', NULL, 30, 60, false, '2021-03-04 05:26:07+00', '2022-07-08 08:50:11+00', false);
INSERT INTO public.web_forumthread VALUES (100, 'Probe Busy 00', 'Probe Busy 00 description', NULL, 30, 60, false, '2021-03-04 05:09:07+00', '2022-07-08 09:07:11+00', false);
INSERT INTO public.web_forumthread VALUES (101, 'Probe Busy 01', 'Probe Busy 01 description', NULL, 30, 60, false, '2021-03-04 05:10:07+00', '2022-07-08 09:06:11+00', false);
INSERT INTO public.web_forumthread VALUES (103, 'Probe Busy 03', 'Probe Busy 03 description', NULL, 30, 60, false, '2021-03-04 05:12:07+00', '2022-07-08 09:04:11+00', false);
INSERT INTO public.web_forumthread VALUES (107, 'Probe Busy 07', 'Probe Busy 07 description', NULL, 30, 60, false, '2021-03-04 05:16:07+00', '2022-07-08 09:00:11+00', false);
INSERT INTO public.web_forumthread VALUES (109, 'Probe Busy 09', 'Probe Busy 09 description', NULL, 30, 60, false, '2021-03-04 05:18:07+00', '2022-07-08 08:58:11+00', false);
INSERT INTO public.web_forumthread VALUES (110, 'Probe Busy 10', 'Probe Busy 10 description', NULL, 30, 60, false, '2021-03-04 05:19:07+00', '2022-07-08 08:57:11+00', false);
INSERT INTO public.web_forumthread VALUES (122, 'Probe Long Thread', 'Probe Long Thread description', NULL, 32, 61, false, '2021-03-04 05:31:07+00', '2026-09-05 08:03:48.279381+00', false);
INSERT INTO public.web_forumthread VALUES (119, 'Probe Busy 19', 'Probe Busy 19 description', NULL, 30, 60, false, '2021-03-04 05:28:07+00', '2022-07-08 08:48:11+00', false);
INSERT INTO public.web_forumthread VALUES (120, 'Probe Busy 20', 'Probe Busy 20 description', NULL, 30, 60, false, '2021-03-04 05:29:07+00', '2022-07-08 08:47:11+00', false);
INSERT INTO public.web_forumthread VALUES (97, 'Probe Thread', 'Probe Thread description', NULL, 30, 55, false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', false);
INSERT INTO public.web_forumthread VALUES (102, 'Probe Busy 02', 'Probe Busy 02 description', NULL, 30, 60, false, '2021-03-04 05:11:07+00', '2022-07-08 09:05:11+00', false);
INSERT INTO public.web_forumthread VALUES (104, 'Probe Busy 04', 'Probe Busy 04 description', NULL, 30, 60, false, '2021-03-04 05:13:07+00', '2022-07-08 09:03:11+00', false);
INSERT INTO public.web_forumthread VALUES (105, 'Probe Busy 05', 'Probe Busy 05 description', NULL, 30, 60, false, '2021-03-04 05:14:07+00', '2022-07-08 09:02:11+00', false);
INSERT INTO public.web_forumthread VALUES (108, 'Probe Busy 08', 'Probe Busy 08 description', NULL, 30, 60, false, '2021-03-04 05:17:07+00', '2022-07-08 08:59:11+00', false);
INSERT INTO public.web_forumthread VALUES (113, 'Probe Busy 13', 'Probe Busy 13 description', NULL, 30, 60, false, '2021-03-04 05:22:07+00', '2022-07-08 08:54:11+00', false);
INSERT INTO public.web_forumthread VALUES (111, 'Probe Busy 11', 'Probe Busy 11 description', NULL, 30, 60, false, '2021-03-04 05:20:07+00', '2022-07-08 08:56:11+00', false);
INSERT INTO public.web_forumthread VALUES (114, 'Probe Busy 14', 'Probe Busy 14 description', NULL, 30, 60, false, '2021-03-04 05:23:07+00', '2022-07-08 08:53:11+00', false);
INSERT INTO public.web_forumthread VALUES (118, 'Probe Busy 18', 'Probe Busy 18 description', NULL, 30, 60, false, '2021-03-04 05:27:07+00', '2022-07-08 08:49:11+00', false);
INSERT INTO public.web_forumthread VALUES (121, 'Probe Deep Thread', 'Probe Deep Thread description', NULL, 30, 61, false, '2021-03-04 05:30:07+00', '2022-07-08 08:46:11+00', false);
INSERT INTO public.web_forumthread VALUES (123, '', '', 197, NULL, NULL, false, '2026-08-30 13:50:08.159477+00', '2026-08-30 13:50:08.159507+00', false);
INSERT INTO public.web_forumthread VALUES (124, '', '', 196, NULL, NULL, false, '2026-08-30 13:50:37.439639+00', '2026-08-30 13:50:37.439667+00', false);
INSERT INTO public.web_forumthread VALUES (116, 'Probe Busy 16', 'Probe Busy 16 description', NULL, 30, 60, false, '2021-03-04 05:25:07+00', '2022-07-08 08:51:11+00', false);
INSERT INTO public.web_forumthread VALUES (125, '', '', 198, NULL, NULL, false, '2021-03-04 05:06:07+00', '2022-07-08 09:10:11+00', false);

INSERT INTO public.web_forumpost VALUES (53, 'Probe Deep reply 1', '2023-09-10 11:47:13+00', '2023-09-10 11:47:13+00', NULL, 49, 121);
INSERT INTO public.web_forumpost VALUES (68, '   ', '2023-09-10 11:48:13+00', '2023-09-10 11:48:13+00', 32, 49, 121);
INSERT INTO public.web_forumpost VALUES (36, 'Probe Busy 08 post 0', '2023-09-10 11:26:13+00', '2023-09-10 11:26:13+00', 30, NULL, 108);
INSERT INTO public.web_forumpost VALUES (50, 'Probe Deep Thread post 1', '2023-09-10 11:44:13+00', '2023-09-10 11:49:13+00', 30, NULL, 121);
INSERT INTO public.web_forumpost VALUES (54, 'Probe Long Thread post 0', '2023-09-10 11:49:13+00', '2023-09-10 11:49:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (55, 'Probe Long Thread post 1', '2023-09-10 11:50:13+00', '2023-09-10 11:50:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (56, 'Probe Long Thread post 2', '2023-09-10 11:51:13+00', '2023-09-10 11:51:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (57, 'Probe Long Thread post 3', '2023-09-10 11:52:13+00', '2023-09-10 11:52:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (58, 'Probe Long Thread post 4', '2023-09-10 11:53:13+00', '2023-09-10 11:53:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (59, 'Probe Long Thread post 5', '2023-09-10 11:54:13+00', '2023-09-10 11:54:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (60, 'Probe Long Thread post 6', '2023-09-10 11:55:13+00', '2023-09-10 11:55:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (22, 'Probe Thread post 0', '2023-09-10 11:12:13+00', '2023-09-10 11:12:13+00', 30, NULL, 97);
INSERT INTO public.web_forumpost VALUES (23, 'Probe Thread post 1', '2023-09-10 11:13:13+00', '2023-09-10 11:13:13+00', 30, NULL, 97);
INSERT INTO public.web_forumpost VALUES (24, 'Probe Thread post 2', '2023-09-10 11:14:13+00', '2023-09-10 11:14:13+00', 30, NULL, 97);
INSERT INTO public.web_forumpost VALUES (37, 'Probe Busy 09 post 0', '2023-09-10 11:27:13+00', '2023-09-10 11:27:13+00', 30, NULL, 109);
INSERT INTO public.web_forumpost VALUES (38, 'Probe Busy 10 post 0', '2023-09-10 11:28:13+00', '2023-09-10 11:28:13+00', 30, NULL, 110);
INSERT INTO public.web_forumpost VALUES (39, 'Probe Busy 11 post 0', '2023-09-10 11:29:13+00', '2023-09-10 11:29:13+00', 30, NULL, 111);
INSERT INTO public.web_forumpost VALUES (61, 'Probe Long Thread post 7', '2023-09-10 11:56:13+00', '2023-09-10 11:56:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (25, 'Probe Locked Thread post 0', '2023-09-10 11:15:13+00', '2023-09-10 11:15:13+00', 32, NULL, 98);
INSERT INTO public.web_forumpost VALUES (40, 'Probe Busy 12 post 0', '2023-09-10 11:30:13+00', '2023-09-10 11:30:13+00', 30, NULL, 112);
INSERT INTO public.web_forumpost VALUES (41, 'Probe Busy 13 post 0', '2023-09-10 11:31:13+00', '2023-09-10 11:31:13+00', 30, NULL, 113);
INSERT INTO public.web_forumpost VALUES (42, 'Probe Busy 14 post 0', '2023-09-10 11:32:13+00', '2023-09-10 11:32:13+00', 30, NULL, 114);
INSERT INTO public.web_forumpost VALUES (43, 'Probe Busy 15 post 0', '2023-09-10 11:33:13+00', '2023-09-10 11:33:13+00', 30, NULL, 115);
INSERT INTO public.web_forumpost VALUES (44, 'Probe Busy 16 post 0', '2023-09-10 11:34:13+00', '2023-09-10 11:34:13+00', 30, NULL, 116);
INSERT INTO public.web_forumpost VALUES (45, 'Probe Busy 17 post 0', '2023-09-10 11:35:13+00', '2023-09-10 11:35:13+00', 30, NULL, 117);
INSERT INTO public.web_forumpost VALUES (46, 'Probe Busy 18 post 0', '2023-09-10 11:36:13+00', '2023-09-10 11:36:13+00', 30, NULL, 118);
INSERT INTO public.web_forumpost VALUES (26, 'Probe Pinned Thread post 0', '2023-09-10 11:16:13+00', '2023-09-10 11:16:13+00', 32, NULL, 99);
INSERT INTO public.web_forumpost VALUES (27, 'Probe Pinned Thread post 1', '2023-09-10 11:17:13+00', '2023-09-10 11:17:13+00', 32, NULL, 99);
INSERT INTO public.web_forumpost VALUES (47, 'Probe Busy 19 post 0', '2023-09-10 11:37:13+00', '2023-09-10 11:37:13+00', 30, NULL, 119);
INSERT INTO public.web_forumpost VALUES (28, 'Probe Busy 00 post 0', '2023-09-10 11:18:13+00', '2023-09-10 11:18:13+00', 30, NULL, 100);
INSERT INTO public.web_forumpost VALUES (29, 'Probe Busy 01 post 0', '2023-09-10 11:19:13+00', '2023-09-10 11:19:13+00', 30, NULL, 101);
INSERT INTO public.web_forumpost VALUES (30, 'Probe Busy 02 post 0', '2023-09-10 11:20:13+00', '2023-09-10 11:20:13+00', 30, NULL, 102);
INSERT INTO public.web_forumpost VALUES (31, 'Probe Busy 03 post 0', '2023-09-10 11:21:13+00', '2023-09-10 11:21:13+00', 30, NULL, 103);
INSERT INTO public.web_forumpost VALUES (32, 'Probe Busy 04 post 0', '2023-09-10 11:22:13+00', '2023-09-10 11:22:13+00', 30, NULL, 104);
INSERT INTO public.web_forumpost VALUES (33, 'Probe Busy 05 post 0', '2023-09-10 11:23:13+00', '2023-09-10 11:23:13+00', 30, NULL, 105);
INSERT INTO public.web_forumpost VALUES (34, 'Probe Busy 06 post 0', '2023-09-10 11:24:13+00', '2023-09-10 11:24:13+00', 30, NULL, 106);
INSERT INTO public.web_forumpost VALUES (35, 'Probe Busy 07 post 0', '2023-09-10 11:25:13+00', '2023-09-10 11:25:13+00', 30, NULL, 107);
INSERT INTO public.web_forumpost VALUES (48, 'Probe Busy 20 post 0', '2023-09-10 11:38:13+00', '2023-09-10 11:38:13+00', 30, NULL, 120);
INSERT INTO public.web_forumpost VALUES (66, 'probe post 0', '2023-09-10 11:39:13+00', '2023-09-10 11:39:13+00', 30, NULL, 17);
INSERT INTO public.web_forumpost VALUES (67, 'probe post 1', '2023-09-10 11:40:13+00', '2023-09-10 11:40:13+00', 32, NULL, 17);
INSERT INTO public.web_forumpost VALUES (62, 'Probe Long Thread post 8', '2023-09-10 11:57:13+00', '2023-09-10 11:57:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (1, 'probe post 0', '2023-09-10 11:41:13+00', '2023-09-10 11:41:13+00', 30, NULL, 14);
INSERT INTO public.web_forumpost VALUES (2, 'probe post 1', '2023-09-10 11:42:13+00', '2023-09-10 11:42:13+00', 30, NULL, 14);
INSERT INTO public.web_forumpost VALUES (63, 'Probe Long Thread post 9', '2023-09-10 11:58:13+00', '2023-09-10 11:58:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (51, 'Probe Deep reply 0', '2023-09-10 11:45:13+00', '2023-09-10 11:45:13+00', 32, 49, 121);
INSERT INTO public.web_forumpost VALUES (64, 'Probe Long Thread post 10', '2023-09-10 11:59:13+00', '2023-09-10 11:59:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (65, 'Probe Long Thread post 11', '2023-09-10 12:00:13+00', '2023-09-10 12:00:13+00', 32, NULL, 122);
INSERT INTO public.web_forumpost VALUES (49, 'Probe Deep Thread post 0', '2023-09-10 11:43:13+00', '2023-09-10 11:43:13+00', 30, NULL, 121);
INSERT INTO public.web_forumpost VALUES (52, 'Probe Deep reply 0 0', '2023-09-10 11:46:13+00', '2023-09-10 11:46:13+00', 30, 51, 121);

INSERT INTO public.web_forumpostversion VALUES (1, 'Probe Thread body 0', 22, '2026-08-29 09:24:05.591432+00', 30);
INSERT INTO public.web_forumpostversion VALUES (2, 'Probe Thread body 1', 23, '2026-08-29 09:24:05.657271+00', 30);
INSERT INTO public.web_forumpostversion VALUES (3, 'Probe Thread body 2', 24, '2026-08-29 09:24:05.66562+00', 30);
INSERT INTO public.web_forumpostversion VALUES (4, 'Probe Locked Thread body 0', 25, '2026-08-29 09:24:05.687944+00', 32);
INSERT INTO public.web_forumpostversion VALUES (5, 'Probe Pinned Thread body 0', 26, '2026-08-30 07:32:06.033939+00', 32);
INSERT INTO public.web_forumpostversion VALUES (6, 'Probe Pinned Thread body 1', 27, '2026-08-30 07:32:06.107797+00', 32);
INSERT INTO public.web_forumpostversion VALUES (7, 'Probe Busy 00 body 0', 28, '2026-08-30 07:32:06.176539+00', 30);
INSERT INTO public.web_forumpostversion VALUES (8, 'Probe Busy 01 body 0', 29, '2026-08-30 07:32:06.198682+00', 30);
INSERT INTO public.web_forumpostversion VALUES (9, 'Probe Busy 02 body 0', 30, '2026-08-30 07:32:06.219216+00', 30);
INSERT INTO public.web_forumpostversion VALUES (10, 'Probe Busy 03 body 0', 31, '2026-08-30 07:32:06.239631+00', 30);
INSERT INTO public.web_forumpostversion VALUES (11, 'Probe Busy 04 body 0', 32, '2026-08-30 07:32:06.258591+00', 30);
INSERT INTO public.web_forumpostversion VALUES (12, 'Probe Busy 05 body 0', 33, '2026-08-30 07:32:06.275+00', 30);
INSERT INTO public.web_forumpostversion VALUES (13, 'Probe Busy 06 body 0', 34, '2026-08-30 07:32:06.290098+00', 30);
INSERT INTO public.web_forumpostversion VALUES (14, 'Probe Busy 07 body 0', 35, '2026-08-30 07:32:06.30823+00', 30);
INSERT INTO public.web_forumpostversion VALUES (15, 'Probe Busy 08 body 0', 36, '2026-08-30 07:32:06.324405+00', 30);
INSERT INTO public.web_forumpostversion VALUES (16, 'Probe Busy 09 body 0', 37, '2026-08-30 07:32:06.339715+00', 30);
INSERT INTO public.web_forumpostversion VALUES (17, 'Probe Busy 10 body 0', 38, '2026-08-30 07:32:06.355843+00', 30);
INSERT INTO public.web_forumpostversion VALUES (18, 'Probe Busy 11 body 0', 39, '2026-08-30 07:32:06.37278+00', 30);
INSERT INTO public.web_forumpostversion VALUES (19, 'Probe Busy 12 body 0', 40, '2026-08-30 07:32:06.389201+00', 30);
INSERT INTO public.web_forumpostversion VALUES (20, 'Probe Busy 13 body 0', 41, '2026-08-30 07:32:06.405398+00', 30);
INSERT INTO public.web_forumpostversion VALUES (21, 'Probe Busy 14 body 0', 42, '2026-08-30 07:32:06.422392+00', 30);
INSERT INTO public.web_forumpostversion VALUES (22, 'Probe Busy 15 body 0', 43, '2026-08-30 07:32:06.437878+00', 30);
INSERT INTO public.web_forumpostversion VALUES (23, 'Probe Busy 16 body 0', 44, '2026-08-30 07:32:06.454572+00', 30);
INSERT INTO public.web_forumpostversion VALUES (24, 'Probe Busy 17 body 0', 45, '2026-08-30 07:32:06.469675+00', 30);
INSERT INTO public.web_forumpostversion VALUES (25, 'Probe Busy 18 body 0', 46, '2026-08-30 07:32:06.486437+00', 30);
INSERT INTO public.web_forumpostversion VALUES (26, 'Probe Busy 19 body 0', 47, '2026-08-30 07:32:06.501679+00', 30);
INSERT INTO public.web_forumpostversion VALUES (27, 'Probe Busy 20 body 0', 48, '2026-08-30 07:32:06.517549+00', 30);
INSERT INTO public.web_forumpostversion VALUES (28, 'probe comment 0', 1, '2026-08-30 09:38:14.750574+00', 30);
INSERT INTO public.web_forumpostversion VALUES (29, 'probe comment 1', 2, '2026-08-30 09:38:14.757717+00', 30);
INSERT INTO public.web_forumpostversion VALUES (30, 'Probe Deep Thread body 0', 49, '2026-08-30 09:38:14.786228+00', 30);
INSERT INTO public.web_forumpostversion VALUES (31, 'Probe Deep Thread body 1', 50, '2026-08-30 09:38:14.793278+00', 30);
INSERT INTO public.web_forumpostversion VALUES (32, 'reply naming @probe-author and @nobody-at-all', 51, '2026-08-30 09:38:14.809879+00', 32);
INSERT INTO public.web_forumpostversion VALUES (33, 'a reply to a reply', 52, '2026-08-30 09:38:14.819144+00', 30);
INSERT INTO public.web_forumpostversion VALUES (34, 'a reply the site itself made', 53, '2026-08-30 09:38:14.828263+00', NULL);
INSERT INTO public.web_forumpostversion VALUES (35, 'Probe Deep Thread body 1 edited', 50, '2026-08-30 09:38:14.836705+00', 32);
INSERT INTO public.web_forumpostversion VALUES (36, 'Probe Long Thread body 0', 54, '2026-08-30 09:38:14.846488+00', 32);
INSERT INTO public.web_forumpostversion VALUES (37, 'Probe Long Thread body 1', 55, '2026-08-30 09:38:14.852578+00', 32);
INSERT INTO public.web_forumpostversion VALUES (38, 'Probe Long Thread body 2', 56, '2026-08-30 09:38:14.858739+00', 32);
INSERT INTO public.web_forumpostversion VALUES (39, 'Probe Long Thread body 3', 57, '2026-08-30 09:38:14.86542+00', 32);
INSERT INTO public.web_forumpostversion VALUES (40, 'Probe Long Thread body 4', 58, '2026-08-30 09:38:14.871697+00', 32);
INSERT INTO public.web_forumpostversion VALUES (41, 'Probe Long Thread body 5', 59, '2026-08-30 09:38:14.87937+00', 32);
INSERT INTO public.web_forumpostversion VALUES (42, 'Probe Long Thread body 6', 60, '2026-08-30 09:38:14.886711+00', 32);
INSERT INTO public.web_forumpostversion VALUES (43, 'Probe Long Thread body 7', 61, '2026-08-30 09:38:14.892853+00', 32);
INSERT INTO public.web_forumpostversion VALUES (44, 'Probe Long Thread body 8', 62, '2026-08-30 09:38:14.898912+00', 32);
INSERT INTO public.web_forumpostversion VALUES (45, 'Probe Long Thread body 9', 63, '2026-08-30 09:38:14.905582+00', 32);
INSERT INTO public.web_forumpostversion VALUES (46, 'Probe Long Thread body 10', 64, '2026-08-30 09:38:14.911426+00', 32);
INSERT INTO public.web_forumpostversion VALUES (47, 'Probe Long Thread body 11', 65, '2026-08-30 09:38:14.91856+00', 32);
INSERT INTO public.web_forumpostversion VALUES (48, 'probe comment 0', 66, '2026-08-30 09:38:14.962083+00', 30);
INSERT INTO public.web_forumpostversion VALUES (49, 'probe comment 1', 67, '2026-08-30 09:38:14.972072+00', 32);
INSERT INTO public.web_forumpostversion VALUES (50, 'a reply whose title is only spaces', 68, '2026-08-30 11:18:50.705935+00', 32);

INSERT INTO public.web_invitelink VALUES (2, 'register', 'link', 'newcomer@example.org', '', 'de6yjt-eebf6e70d8fa0567329d0a7ebb81f24c', 'MTkx', '2026-08-31 12:07:05.290736+00', '2026-08-31 12:07:17.678379+00', 'newcomer', 190, 191);
INSERT INTO public.web_invitelink VALUES (3, 'claim', 'link', '', 'probe-wd-original', 'de6yoi-012bce977bc3668fed22801e18b609c1', 'MzE', '2026-08-31 12:09:54.485716+00', NULL, '', 190, 31);

INSERT INTO public.web_site VALUES (1, 'wikit', 'Test Wiki', '差分测试用', '', 'localhost', 'media.localhost', 'main', NULL, NULL, 3, '本页内容采用 [http://creativecommons.org/licenses/by-sa/3.0/ CC BY-SA 3.0] 授权。© [[module time]]%%currentyear%%[[/module]] //Probe Wiki//', 'letmein', true, 4, '普通注册后自动获得「读者」；认领 Wikidot 账号后获得「成员」', 4, NULL, '', 'optional');

INSERT INTO public.web_settings VALUES (1, 'default', NULL, 1, 'default');
INSERT INTO public.web_settings VALUES (2, 'stars', 1, NULL, 'default');
INSERT INTO public.web_settings VALUES (82, 'disabled', 96, NULL, 'default');

INSERT INTO public.web_usedtoken VALUES (2, 'de6yjt-eebf6e70d8fa0567329d0a7ebb81f24c', true);

INSERT INTO public.web_user_roles VALUES (203, 189, 4);
INSERT INTO public.web_user_roles VALUES (205, 191, 3);

INSERT INTO public.web_usernotification VALUES (1, 'welcome', '{"probe": "welcome"}', '2026-08-26 11:46:46.995945+00');
INSERT INTO public.web_usernotification VALUES (3, 'welcome', '{}', '2026-08-31 12:02:19.153799+00');
INSERT INTO public.web_usernotification VALUES (4, 'welcome', '{}', '2026-08-31 12:07:17.691244+00');

INSERT INTO public.web_usernotificationmapping VALUES (1, false, 1, 30);
INSERT INTO public.web_usernotificationmapping VALUES (3, false, 3, 31);
INSERT INTO public.web_usernotificationmapping VALUES (4, false, 4, 191);

INSERT INTO public.web_usernotificationsubscription VALUES (1, NULL, 1, 1);
INSERT INTO public.web_usernotificationsubscription VALUES (2, NULL, 2, 1);
INSERT INTO public.web_usernotificationsubscription VALUES (3, NULL, 3, 1);
INSERT INTO public.web_usernotificationsubscription VALUES (4, NULL, 4, 1);
INSERT INTO public.web_usernotificationsubscription VALUES (5, NULL, 5, 1);
INSERT INTO public.web_usernotificationsubscription VALUES (6, 12, NULL, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (7, NULL, 14, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (8, NULL, 16, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (9, NULL, 17, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (10, NULL, 18, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (11, NULL, 43, 30);
INSERT INTO public.web_usernotificationsubscription VALUES (12, NULL, 125, 30);

INSERT INTO public.web_userticket VALUES (2, 'membershipapply', 'Ïë¼ÓÈë±à¼­×é', 'ÎÒ´òËãÕûÀíÒ»Åú·ÖÀàÒ³¡£', 'pwikit-demo', 'pending', '', '2026-08-31 11:10:32.824125+00', NULL, 189, NULL, NULL);
INSERT INTO public.web_userticket VALUES (3, 'ticket', 'Í¼Æ¬¹ÒÁË', 'Ê×Ò³µÚÈýÕÅÍ¼ 404 ÁË¡£', 'pwikit-demo', 'pending', '', '2026-08-31 11:10:32.988892+00', NULL, 189, NULL, NULL);
INSERT INTO public.web_userticket VALUES (4, 'membershipapply', 'Èë×éÉêÇë', '¹Ì¶¨±êÌâµÄÄÇ¸ö±íµ¥¡£', 'pwikit-demo', 'pending', '', '2026-08-31 11:21:02.499409+00', NULL, 189, NULL, NULL);
INSERT INTO public.web_userticket VALUES (5, 'ticket', '', 'Ã»ÓÐ±êÌâµÄÄÇ¸ö±íµ¥¡£', 'pwikit-demo', 'pending', '', '2026-08-31 11:21:02.621146+00', NULL, 189, NULL, NULL);

INSERT INTO public.web_vote VALUES (1, 1, 12, 30, '2026-08-31 07:03:13.054324+00', NULL);
INSERT INTO public.web_vote VALUES (2, 1, 12, 31, '2026-08-31 07:03:13.060778+00', NULL);
INSERT INTO public.web_vote VALUES (3, -1, 12, 32, '2026-08-31 07:03:13.065433+00', NULL);
INSERT INTO public.web_vote VALUES (4, 4, 14, 30, '2026-08-31 07:03:13.068942+00', NULL);
INSERT INTO public.web_vote VALUES (5, 5, 14, 32, '2026-08-31 07:03:13.0724+00', NULL);
INSERT INTO public.web_vote VALUES (89, 3, 139, 30, '2026-08-31 07:03:13.076411+00', NULL);
INSERT INTO public.web_vote VALUES (90, 3, 139, 32, '2026-08-31 07:03:13.093309+00', NULL);
INSERT INTO public.web_vote VALUES (115, 1, 5, 67, '2026-08-30 09:09:18.003133+00', NULL);
INSERT INTO public.web_vote VALUES (91, 4, 139, 31, '2026-08-31 07:03:13.097853+00', NULL);
INSERT INTO public.web_vote VALUES (87, 0.5, 137, 30, '2026-08-31 07:03:13.101847+00', NULL);
INSERT INTO public.web_vote VALUES (88, 1, 137, 32, '2026-08-31 07:03:13.106144+00', NULL);
INSERT INTO public.web_vote VALUES (6, 1, 15, 33, '2026-08-31 07:03:13.111282+00', NULL);
INSERT INTO public.web_vote VALUES (7, -1, 15, 34, '2026-08-31 07:03:13.125237+00', NULL);
INSERT INTO public.web_vote VALUES (8, -1, 15, 35, '2026-08-31 07:03:13.12927+00', NULL);
INSERT INTO public.web_vote VALUES (9, -1, 15, 36, '2026-08-31 07:03:13.134001+00', NULL);
INSERT INTO public.web_vote VALUES (10, -1, 15, 37, '2026-08-31 07:03:13.138064+00', NULL);
INSERT INTO public.web_vote VALUES (11, -1, 15, 38, '2026-08-31 07:03:13.141783+00', NULL);
INSERT INTO public.web_vote VALUES (12, -1, 15, 39, '2026-08-31 07:03:13.145463+00', NULL);
INSERT INTO public.web_vote VALUES (13, -1, 15, 40, '2026-08-31 07:03:13.153034+00', NULL);

SELECT pg_catalog.setval('public.auth_group_id_seq', 1, false);

SELECT pg_catalog.setval('public.auth_group_permissions_id_seq', 1, false);

SELECT pg_catalog.setval('public.dynamic_preferences_globalpreferencemodel_id_seq', 1, false);

SELECT pg_catalog.setval('public.dynamic_preferences_users_userpreferencemodel_id_seq', 30, true);

SELECT pg_catalog.setval('public.web_actionlogentry_id_seq', 26, true);

SELECT pg_catalog.setval('public.web_article_authors_id_seq', 54, true);

SELECT pg_catalog.setval('public.web_article_id_seq', 308, true);

SELECT pg_catalog.setval('public.web_article_tags_id_seq', 13, true);

SELECT pg_catalog.setval('public.web_articlefavourite_id_seq', 8, true);

SELECT pg_catalog.setval('public.web_articlelogentry_id_seq', 355, true);

SELECT pg_catalog.setval('public.web_articlesearchindex_id_seq', 46, true);

SELECT pg_catalog.setval('public.web_articleversion_id_seq', 133, true);

SELECT pg_catalog.setval('public.web_category_id_seq', 156, true);

SELECT pg_catalog.setval('public.web_category_permissions_override_id_seq', 20, true);

SELECT pg_catalog.setval('public.web_directmessage_id_seq', 1, true);

SELECT pg_catalog.setval('public.web_directmessageblock_id_seq', 1, true);

SELECT pg_catalog.setval('public.web_externallink_id_seq', 93, true);

SELECT pg_catalog.setval('public.web_file_id_seq', 1, true);

SELECT pg_catalog.setval('public.web_forumcategory_id_seq', 61, true);

SELECT pg_catalog.setval('public.web_forumpost_id_seq', 71, true);

SELECT pg_catalog.setval('public.web_forumpostlike_id_seq', 82, true);

SELECT pg_catalog.setval('public.web_forumpostversion_id_seq', 53, true);

SELECT pg_catalog.setval('public.web_forumsection_id_seq', 73, true);

SELECT pg_catalog.setval('public.web_forumthread_id_seq', 125, true);

SELECT pg_catalog.setval('public.web_invitelink_id_seq', 3, true);

SELECT pg_catalog.setval('public.web_role_restrictions_id_seq', 10, true);

SELECT pg_catalog.setval('public.web_rolepermissionsoverride_id_seq', 20, true);

SELECT pg_catalog.setval('public.web_rolepermissionsoverride_permissions_id_seq', 10, true);

SELECT pg_catalog.setval('public.web_rolepermissionsoverride_restrictions_id_seq', 10, true);

SELECT pg_catalog.setval('public.web_settings_id_seq', 82, true);

SELECT pg_catalog.setval('public.web_site_id_seq', 1, false);

SELECT pg_catalog.setval('public.web_tag_id_seq', 6, true);

SELECT pg_catalog.setval('public.web_tagscategory_id_seq', 3, true);

SELECT pg_catalog.setval('public.web_usedtoken_id_seq', 2, true);

SELECT pg_catalog.setval('public.web_user_groups_id_seq', 1, false);

SELECT pg_catalog.setval('public.web_user_id_seq', 244, true);

SELECT pg_catalog.setval('public.web_user_roles_id_seq', 215, true);

SELECT pg_catalog.setval('public.web_user_user_permissions_id_seq', 1, false);

SELECT pg_catalog.setval('public.web_usernotification_id_seq', 34, true);

SELECT pg_catalog.setval('public.web_usernotificationmapping_id_seq', 34, true);

SELECT pg_catalog.setval('public.web_usernotificationsubscription_id_seq', 18, true);

SELECT pg_catalog.setval('public.web_userreport_id_seq', 1, false);

SELECT pg_catalog.setval('public.web_userticket_id_seq', 5, true);

SELECT pg_catalog.setval('public.web_vote_id_seq', 116, true);

