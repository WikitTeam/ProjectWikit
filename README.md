<div align="center">
  <h1>ProjectWikit</h1>
  <h3>ProjectWikit维基启动器，基于SCP-RU使用的引擎修改而来，维基语法转换的部分基于SCP基金会社区WikiJump团队开发的FTML。</h3>
</div>




> [!NOTE]
> This project, ProjectWikit, originated as a fork of the [RuFoundation Engine](https://github.com/SCPRu/RuFoundation). However, It has since undergone numerous modifications and has become an independent project, no longer tracking or following updates from the original RuFoundation codebase.
> 
> As a result, ProjectWikit is fully maintained by its own development team. If you encounter any issues or have suggestions, please report them directly to the WikitTeam rather than the original RuFoundation Team.
>
> -----
> ProjectWikit最初是 [RuFoundation引擎](https://github.com/SCPRu/RuFoundation) 的分支。然而，自此之后，它已历经了大量的修改，并已成为独立的项目，不再追踪或跟随原始的RuFoundation代码库的更新。
> 
> 因此，ProjectWikit完全由其自身的开发团队维护。如果您遇到任何问题或有任何建议，请直接向WikitTeam报告，而不是向原始的RuFoundation团队反馈。

## 🖥️ 环境要求
以下是测试时的环境，你可能会与之有出入
- Windows 10
- PostgreSQL 17.2
- NodeJS v17.3.0
- Python 3.13.2
- Rust 1.63

以下为 PostgreSQL 的默认配置，你可以通过给定的环境变量进行修改：
| 名称      | 变量名          | 变量值               |
| :-------- | :-------------- | :------------------ |
| 用户名     | `admin`         | `POSTGRES_USER`     |
| 密码       | `wikitpassword` | `POSTGRES_PASSWORD` |
| 数据库名   | `projwikit`     | `POSTGRES_DB`       |
| 数据库主机 | `localhost`     | `DB_PG_HOST`        |
| 数据库端口 | `5432`          | `DB_PG_PORT`        |

## 💡 快速部署
<details>
<summary><code><strong>使用 Docker 快速部署（推荐）</strong></code></summary>

### 1️⃣ Docker环境
- Docker 28.4.0
- Docker Compose 2.39.4

### 2️⃣ Docker 部署
   - **【STEP1】** 启动项目，运行 `docker compose up`
  
   - **【STEP2】** 在数据库中创建用户、网站并填充初始数据，请先启动项目，然后使用如下命令：
     - `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d localhost -t "网站标题" -H "网站副标题"`
     - `docker exec -it wikitgo-web-1 seed`
    
   - **【OTHER】** 对于无法使用上述指令的场合：
     - `docker compose up`
     - `docker exec -it wikitgo-web-1 python manage.py migrate`
     - `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d 网站域名 -t "网站标题" -H "网站副标题"`
     - `docker exec -it wikitgo-web-1 python manage.py createsuperuser`
     - `docker exec -it wikitgo-web-1 python manage.py seed`

### *️⃣ 其他操作
   - 完全删除所有数据：
     - `docker compose down`
     - `rm -rf ./files ./archive ./postgresql`

   - 要更新正在运行的应用：
     - `docker compose up -d --no-deps --build web`
</details>

<details>
<summary><code><strong>使用 Python 快速部署</strong></code></summary>

### 1️⃣ Python环境
- Python 3.13.2

### 2️⃣ Python 部署
   - **【STEP1】** 安装配置环境
     - 定位到 `web/js` 目录，执行 `yarn install`
     - 在项目根目录下，运行 `pip install -r requirements.txt`
      
   - **【STEP2】** 启动配置
       - `python manage.py migrate`
       - `python manage.py runserver --watch`
    
   - **【STEP3】** 创建管理员账户
       - 运行 `python manage.py createsuperuser --username Admin --email "" --skip-checks`
       - 根据终端提示完成操作
    
   - **【STEP4】**  数据库初始数据填充
       - 创建以下基础对象：
         - 网站记录（用于本地主机）
         - 部分重要的页面（如 `nav:top` 或 `nav:side`）
       - 或者，通过运行以下命令来配置这些基本结构：
         - `python manage.py createsite -s wikit-wiki -d localhost -t "网站标题" -H "网站副标题"`
         - `python manage.py seed`
 
</details>
