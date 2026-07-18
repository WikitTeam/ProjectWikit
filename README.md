<div align="center">
  <h1>ProjectWikit</h1>
  <h3>旨在将Wikidot站点完整地迁移到兼容Wikidot结构的ProjectWikit，并支持基于FTML的Wikidot语法。</h3>
</div>

> [!NOTE]
> This project, ProjectWikit, originated as a fork of the [RuFoundation Engine](https://github.com/SCPRu/RuFoundation). However, it has since undergone numerous modifications, structural changes, and feature enhancements, becoming an independent project, no longer tracking or following updates from the original RuFoundation codebase, and no longer maintaining compatibility with it.
> 
> As a result, ProjectWikit is fully maintained by its own development team. If you encounter any issues or have suggestions, please report them directly to the WikitTeam rather than the original RuFoundation Team.
>
> -----
> ProjectWikit最初是 [RuFoundation引擎](https://github.com/SCPRu/RuFoundation) 的分支。然而，自此之后，它已历经了大量的修改、结构调整和功能扩展，并已成为一个独立的项目，不再追踪或跟随原始的RuFoundation代码库的更新，亦不再保持对其的兼容。
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
> 在开始部署前，请**复制** `.env.example` 为 `.env`。数据库名称/密码、对外端口、栈名等所有部署配置都集中在 `.env` 里，无需再改 `docker-compose.yaml`。

`.env` 示例：

```dotenv
# ---- 部署 ----
COMPOSE_PROJECT_NAME=wikitgo
WEB_PORT=8000

# ---- 自动更新 ----
HOST_PROJECT_DIR=/opt/ProjectWikit
UPDATE_REPO=WikitTeam/ProjectWikit
UPDATE_BRANCH=master
UPDATE_POLL_INTERVAL=600

# ---- 数据库 ----
DB_PG_DATABASE=projwikit
DB_PG_USERNAME=admin
DB_PG_PASSWORD=改成你的密码

# ---- Django ----
SECRET_KEY=改成一段足够长的随机字符串
DEBUG=false
```

> [!IMPORTANT]
> 如需使用「后台自动更新」，`HOST_PROJECT_DIR` 必须设为本项目在**宿主机**上的绝对路径（在项目根目录执行 `pwd` 查看，例如 `/opt/ProjectWikit`），否则 updater 容器无法拉取代码与重建。

<details>
<summary>  <code><strong>使用 Docker 快速部署（推荐）</strong></code></summary>

### 1.Docker环境
- Docker 28.4.0
- Docker Compose 2.39.4

### 2.Docker 部署
   - **【STEP1】** 启动项目，运行 `docker compose up`
  
   - **【STEP2】** 在数据库中创建用户、网站并填充初始数据，请先启动项目，然后使用如下命令：
     - `docker exec -it wikitgo-web-1 python manage.py createsite -s wikit-wiki -d 网站域名(本地填写localhost) -t "网站标题" -H "网站副标题"`
     - `docker exec -it wikitgo-web-1 python manage.py migrate`
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

## 自动更新

拥有「管理系统更新」权限的用户可在后台一键把站点更新到 GitHub 上的最新发布版本，无需登录服务器手动操作。

### 功能
- 后台自动定时检查 GitHub 仓库的最新 release（默认每 10 分钟一次），有新版本时在后台提示。
- 展示当前版本、最新版本号与该版本的更新说明。
- 一键更新：自动拉取对应版本代码 → 重建并重启 `web` 服务 → 更新完成后自动清理无用的镜像与旧构建缓存。
- 更新过程实时显示进度与日志；更新失败会给出原因，且不会覆盖服务器上任何用户数据或本地改动。
- 权限受「管理系统更新」（`manage_updates`）控制，可在后台「角色」中授予指定角色。

### 用法
1. 部署前在 `.env` 中正确设置 `HOST_PROJECT_DIR`（见上方「快速部署」）。
2. 正常 `docker compose up -d` 启动，`updater` 服务会随之一起运行，之后全程自动，无需额外配置。
3. 在 GitHub 仓库的 `master` 分支上发布（release）新版本。
4. 用有权限的账户访问 **`/-/admin/update`**，看到「有可用更新」后点击「更新到最新版本」，等待进度完成即可。

> [!NOTE]
> 更新只会在服务器代码可干净切换时进行。若服务器上对被 git 跟踪的文件做过本地改动，更新会中止并提示，以免覆盖你的修改——请把服务器特有配置都放在 `.env`（不被 git 跟踪）中。

## 站点设置与主题

站点的各项参数都在后台 **`/-/admin` → 站点** 面板中设置，改动即时生效，无需改动代码或重新部署。

### 站点面板字段
| 字段 | 说明 |
| --- | --- |
| 缩写 | 站点内部标识符 |
| 标题 / 副标题 | wiki 显示的名称与其下方的小字 |
| 图标 | 站点图标 |
| 文章域名（主域名） | 访问 wiki 页面所用的主域名 |
| 文件域名 | 提供用户上传文件/附件的域名，可与主域名不同以做隔离 |
| 主页名称 | 访问站点根路径（`/`）时展示哪个页面，默认 `main` |
| 站点主题 | 选择当前启用的主题（见下方「主题」） |
| 评分系统 | 可选 默认 / 禁用 / 点赞·点踩 / 星级评分；默认即点赞·点踩（uv/dv）制 |
| 用户是否可以创建标签 | `默认` / `禁止` / `允许` |

### 主页设定
在「站点」面板的 **主页名称** 填入页面名即可把该页面设为首页（默认 `main`）。例如填 `start`，则访问根路径会显示 `start` 这篇文章。

### 主题
主题允许你在后台直接编辑站点 CSS，无需修改仓库里的文件。

- 在后台 **`/-/admin` → 主题** 中可以创建多个主题，每个主题取一个名字，并二选一：**内联 CSS** 或 **外部链接**。
- 在「站点」面板的 **站点主题** 下拉中选择要启用的那一个即可切换全站外观。
- 未选择任何主题时，回退到项目自带的默认样式。

> [!TIP]
> 首次部署后系统会自动生成一个名为「默认主题」的主题，其内容即当前的默认样式，可在它基础上修改，或复制一份另存为新主题。

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
| `--update-existing` | 对库中已存在的文章也重新同步标签与评分（默认已存在文章整体跳过）。仅补充缺失的评分，不会覆盖已有投票 |

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

   - **页面已经导入过，只想补上标签和评分**：追加 `--update-existing`
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive -s pages -t --update-existing`

   - **附件 / 图片显示 not found（物理文件缺失）**：重跑一次页面迁移即可，脚本会检测缺失并从备份补拷
     - `docker exec -it wikitgo-web-1 python manage.py seed -a ./archive -s pages`

   - **迁移后搜索不到文章**（搜索索引未建）：运行 `initsearch` 重建全站搜索索引
     - `docker exec -it wikitgo-web-1 python manage.py initsearch`

> [!NOTE]
> `forum` 依赖页面数据来关联文章的评论区，因此单独迁移论坛前，请确保对应页面已经迁移完成；使用 `all` 时无需担心，脚本会先迁移页面再迁移论坛。

> [!NOTE]
> 迁移脚本不会自动建立搜索索引（索引仅在编辑文章时更新）。因此**每次迁移完成后都应运行一次 `initsearch`**，否则新导入的文章无法被站内搜索检索到。

> [!NOTE]
> 文件迁移是增量安全的：不会清空已有的媒体目录，重复运行只会补拷缺失的物理文件，不会删除或重复已存在的附件。因此附件出现 not found 时，直接重跑页面迁移即可恢复（前提是备份中的 `files/` 目录仍然完整）。
