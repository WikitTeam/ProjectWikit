<div align="center">
  <h1>ProjectWikit</h1>
  <h3>ProjectWikit维基启动器，基于SCP-RU使用的引擎更改而来，维基语法转换的部分基于SCP基金会社区WikiJump团队开发的FTML。</h3>
</div>

### Note From WikitTeam

This project, ProjectWikit, originated as a fork of the RuFoundation Engine. (Link to this: https://github.com/SCPRu/RuFoundation) However, It has since undergone numerous modifications and has become an independent project, no longer tracking or following updates from the original RuFoundation codebase.

As a result, ProjectWikit is fully maintained by its own development team. If you encounter any issues or have suggestions, please report them directly to the WikitTeam rather than the original RuFoundation Team.

ProjectWikit最初是RuFoundation引擎（项目链接：https://github.com/SCPRu/RuFoundation ）的一个分支。然而，自此之后，它已经经历了大量的修改，并已成为一个独立的项目，不再追踪或跟随原始RuFoundation代码库的更新。

因此，ProjectWikit完全由其自身的开发团队维护。如果您遇到任何问题或有任何建议，请直接向WikitTeam报告，而不是向原始的RuFoundation团队反馈。
## 环境要求（以下是测试时的环境，你可能会与之有出入）

- Windows 10
- PostgreSQL 17.2
- NodeJS v17.3.0
- Python 3.13.2
- Rust 1.63

## PostgreSQL 配置
默认配置为：
- 用户名：`admin` (`POSTGRES_USER`)
- 密码：`wikitpassword` (`POSTGRES_PASSWORD`)
- 数据库名：`projwikit` (`POSTGRES_DB`)
- 数据库主机：`localhost` (`DB_PG_HOST`)
- 数据库端口：`5432` (`DB_PG_PORT`)

您可以通过给定的环境变量进行修改。

## 启动方法

- 首先进入 `web/js` 目录，执行 `yarn install`
- 之后，在项目根目录下运行：
  - `pip install -r requirements.txt`
  - `python manage.py migrate`
  - `python manage.py runserver --watch`

## 创建管理员账户

- 运行 `python manage.py createsuperuser --username Admin --email "" --skip-checks`
- 根据终端提示完成操作

## 数据库初始数据填充

要开始工作，需要以下基础对象：

- 网站记录（用于本地主机）
- 一些对正确显示至关重要的页面（如 `nav:top` 或 `nav:side`）

您可以通过运行以下命令来配置这些基本结构：

- `python manage.py createsite -s wikit-wiki -d localhost -t "网站标题" -H "网站副标题"`
- `python manage.py seed`


## 在Docker中运行（推荐）

### 环境要求（测试版本）：

- Docker 28.4.0
- Docker Compose 2.39.4

### 快速开始

启动项目：

- `docker compose up`

要完全删除所有数据：

- `docker compose down`
- `rm -rf ./files ./archive ./postgresql`

要在数据库中创建用户、网站并填充初始数据，请先启动项目，然后使用如下命令：

- `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d localhost -t "网站标题" -H "网站副标题"`
- `docker exec -it wikitgo-web-1 seed`

对于无法使用上述指令的场合：

- `docker compose up`
- `docker exec -it wikitgo-web-1 python manage.py migrate`
- `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d 网站域名 -t "网站标题" -H "网站副标题"`
- `docker exec -it wikitgo-web-1 python manage.py createsuperuser`
- `docker exec -it wikitgo-web-1 python manage.py seed`

要更新正在运行的应用：

- `docker compose up -d --no-deps --build web`
