<div align="center">
  <h1>ProjectWikit</h1>
  <h3>旨在将Wikidot站点完整地迁移到兼容Wikidot结构的ProjectWikit，并支持基于FTML的Wikidot语法。</h3>
</div>




> [!NOTE]
> This project, ProjectWikit, originated as a fork of the [RuFoundation Engine](https://github.com/SCPRu/RuFoundation). However, It has since undergone numerous modifications and has become an independent project, no longer tracking or following updates from the original RuFoundation codebase.
> 
> As a result, ProjectWikit is fully maintained by its own development team. If you encounter any issues or have suggestions, please report them directly to the WikitTeam rather than the original RuFoundation Team.
>
> -----
> ProjectWikit最初是 [RuFoundation引擎](https://github.com/SCPRu/RuFoundation) 的分支。然而，自此之后，它已历经了大量的修改，并已成为独立的项目，不再追踪或跟随原始的RuFoundation代码库的更新。
> 
> 因此，ProjectWikit完全由其自身的开发团队维护。如果您遇到任何问题或有任何建议，请直接向负责人Kakushi或WikitTeam报告，而不是向原始的RuFoundation团队反馈。

## 环境要求
以下是测试时的环境，你可能会与之有出入
- Windows 10
- PostgreSQL 17.2
- NodeJS v17.3.0
- Python 3.13.2
- Rust 1.63

以下为 PostgreSQL 的默认配置，你可以通过给定的环境变量进行修改：
| 名称      | 变量值          | 变量名               |
| :-------- | :-------------- | :------------------ |
| 用户名     | `admin`         | `POSTGRES_USER`     |
| 密码       | `wikitpassword` | `POSTGRES_PASSWORD` |
| 数据库名   | `projwikit`     | `POSTGRES_DB`       |
| 数据库主机 | `localhost`     | `DB_PG_HOST`        |
| 数据库端口 | `5432`          | `DB_PG_PORT`        |

## 快速部署
> [!TIP]
> 在开始部署前，请修改 `prod-web.env.example` 内容，并将文件重命名为 `prod-web.env`。

<details>
<summary>  <code><strong>使用 Docker 快速部署（推荐）</strong></code></summary>

### 1.Docker环境
- Docker 28.4.0
- Docker Compose 2.39.4

### 2.Docker 部署
   - **【STEP1】** 启动项目，运行 `docker compose up`
  
   - **【STEP2】** 在数据库中创建用户、网站并填充初始数据，请先启动项目，然后使用如下命令：
     - `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d 网站域名(本地填写localhost) -t "网站标题" -H "网站副标题"`
     - `docker exec -it wikitgo-web-1 python manage.py createsuperuser`
     - `docker exec -it wikitgo-web-1 seed`
    
   - **【OTHER】** 对于无法使用上述指令的场合（因权限不足\迁移未完成等造成的问题）：
     - `docker compose down` 若已运行容器，则先关闭容器
     - `sudo chmod -R 777 ./files`
     - `docker compose up -d`
     - `docker exec -it wikitgo-web-1 python manage.py migrate`
     - `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d 网站域名 -t "网站标题" -H "网站副标题"`
     - `docker exec -it wikitgo-web-1 python manage.py createsuperuser`
     - `docker exec -it wikitgo-web-1 python manage.py seed`

### 3.其他操作
   - 从备份完整迁移Wikidot网站：详见下方 [从备份迁移数据](#从备份迁移数据) 章节。

   - 完全删除所有数据：
     - `docker compose down`
     - `rm -rf ./files ./archive ./postgresql`

   - 要更新正在运行的应用：
     - `docker compose up -d --no-deps --build web`
</details>

<details>
<summary>  <code><strong>使用 Python 快速部署</strong></code></summary>

### 1.Python环境
- Python 3.13.2

### 2.Python 部署
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

## 从备份迁移数据
将 [wikitCLI](https://github.com/kakushi-w/wikit) 生成的 Wikidot 备份导入到已部署的站点，包括页面、历史修订、附件、标签、页面评分、父子关系以及论坛。

> [!TIP]
> 迁移前请确认目标站点已创建（即已执行过 `createsite`）。

### 1.准备备份
将备份中的 `_users`、`files`、`forum`、`meta`、`pages` 文件夹一并放入 `./archive` 目录。

### 2.执行迁移
基本命令为 `docker exec -it wikitgo-web-1 python manage.py seed -a <备份路径>`，通过以下参数控制迁移范围与行为。

| 参数 | 作用 |
| :-- | :-- |
| `-a, --archive <路径>` | 指定备份归档路径，启用迁移模式 |
| `-s, --scope {all,pages,forum}` | 迁移范围：`all` 全部、`pages` 仅页面（含文件 / 标签 / 评分 / 父页面）、`forum` 仅论坛。默认 `all` |
| `-t, --force-tags` | 强制迁移标签：当站点设置为“禁止用户创建标签”时，仍然导入备份中的标签 |
| `--no-votes` | 跳过页面评分的迁移 |

### 3.常见场景
   - **完整迁移整个站点**（页面在前、论坛在后，自动按顺序处理）：
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive`

   - **只迁移页面**（例如先导入内容，稍后再单独处理论坛）：
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive -s pages`

   - **只迁移论坛**（页面此前已经导入过）：
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive -s forum`

   - **标签没有被导入**（站点默认禁止创建标签时会出现）：追加 `-t` 强制导入标签
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive -t`

   - **不想导入历史评分**：追加 `--no-votes`
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive --no-votes`

> [!NOTE]
> `forum` 依赖页面数据来关联文章的评论区，因此单独迁移论坛前，请确保对应页面已经迁移完成；使用 `all` 时无需担心，脚本会先迁移页面再迁移论坛。
