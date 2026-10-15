# Quick start

> This is a copy of the Quick start page from the documentation site. For the full text and the pages it links to, see the [Quick start](https://pwikit-docs.unitreaty.org/en/quickstart) on the documentation site.

This page lists only the steps needed to get one site running. Options, alternatives and troubleshooting for each step are in [Deployment](https://pwikit-docs.unitreaty.org/en/deployment).

## Before you start

- A Linux, macOS or Windows machine. On macOS, an ordinary account rather than root.
- A domain name pointing at the machine, and a second one for uploaded files. Both need an A record for the machine's public address.
- For HTTPS, ports 80 and 443 reachable from the internet.

No database setup is needed: pwikit carries PostgreSQL.

## Four steps

1. **Get pwikit** and change into its directory. That directory is the data directory, so `files/`, `pgdata/` and the rest are created there.

   The install script installs into the current directory, which must be empty. On Linux and macOS:

   ```sh
   mkdir /opt/pwikit && cd /opt/pwikit
   curl -fsSL https://github.com/WikitTeam/ProjectWikit/releases/latest/download/install.sh | sh
   ```

   On Windows, run `irm https://github.com/WikitTeam/ProjectWikit/releases/latest/download/install.ps1 | iex` in PowerShell. You can also download a package from GitHub Releases and unpack it yourself.

   When GitHub cannot be reached, install from the mirror instead. The script downloads from the mirror and writes it into `pwikit.toml`, so later updates use it too:

   ```sh
   curl -fsSL https://wikit.unitreaty.org/projwikit/update/latest/download/install.sh | sh
   ```

   On Windows, run `irm https://wikit.unitreaty.org/projwikit/update/latest/download/install.ps1 | iex`. For other ways to get pwikit, see [Deployment](https://pwikit-docs.unitreaty.org/en/deployment#getting-pwikit).

   > [!TIP]
   > When you unpack a package yourself, typing `pwikit` on its own does not work until step 4. However pwikit was installed, steps 2 and 3 work from this directory as `./pwikit` (`.\pwikit.exe` on Windows).

2. **Create the site.** This command creates the database and its tables itself, so the server does not have to run first:

   ```sh
   ./pwikit createsite -slug main -title "My Wiki" -headline "A Wikidot-compatible wiki" \
     -domain wiki.example.org -media-domain files.example.org
   ```

   Replace both domains with your own. To try pwikit on your own computer, use `-domain localhost` and open `http://localhost:8080`.

3. **Fill the site and create the administrator.** Take one of the two paths, depending on where the site comes from.

   **A new, empty site**, with the starter pages:

   ```sh
   ./pwikit seed
   ./pwikit admin create -name "YourName"
   ```

   **An import from Wikidot**, in this order:

   1. Put the unpacked [wikitCLI](https://github.com/kakushi-w/wikit) backup into `archive/` in the data directory, which step 2 created. It should look like this:

      ```text
      archive/
        _users/              the accounts, a set of .json files
        xxx-wiki/            a site directory; its name is the site identifier
          meta/site.json     the site's details; this file makes it a site
          meta/pages/        each page's title, tags, revisions and ratings
          pages/             the page sources, one .7z per page
          files/             the attachments
          forum/             the forum
      ```

      `_users/` has to sit next to the site directory. Without it the imported history, ratings and comments have no authors. For several sites, put every site directory side by side in `archive/`, sharing one `_users/`. The full layout of a backup is described in [The backup](https://pwikit-docs.unitreaty.org/en/importing#the-backup).

   2. Import. Without a directory, `pwikit import` reads `archive/`:

      ```sh
      ./pwikit import
      ```

      When `archive/` holds several sites, add `-from <site directory name>` to choose one. The `N accounts` in the summary at the end should be close to the number of accounts in the backup. When `_users/` cannot be found, the import refuses to run and writes nothing.

   3. Once the import has finished, create the administrator with your own Wikidot username as `-name`, so the imported account and its pages belong to you:

      ```sh
      ./pwikit admin create -name "YourWikidotName"
      ```

   `admin create` asks for a password without displaying it. The account holds every permission on every site of the instance.

   > [!TIP]
   > Taking a backup before the import, how tags and ratings are handled, what running an import again does, and what each error means are in [Importing from Wikidot](https://pwikit-docs.unitreaty.org/en/importing).

4. **Install it as a system service** so it starts at boot. **This is also the step that makes `pwikit` runnable by name from any directory**:

   ```sh
   sudo ./pwikit service install
   sudo ./pwikit service start
   ```

   On Windows, run `.\pwikit.exe service install` and `.\pwikit.exe service start` in an administrator terminal. The first start unpacks and initialises PostgreSQL, so it takes longer than later ones; `pwikit service status` says whether it is running.

   Running `./pwikit serve` in the foreground works too, but it stops when the terminal closes; in that case run `./pwikit path install` to be able to type `pwikit` on its own.

   A service installed by root runs as the dedicated account `pwikit`, without root rights. See [Running as a system service](https://pwikit-docs.unitreaty.org/en/deployment#running-as-a-system-service).

When the site's domain is a public one, pwikit obtains and renews the HTTPS certificate itself and listens on ports 80 and 443.

## After that

> [!IMPORTANT]
> Soon after signing in, set **Role after sign-up** and **Role after claiming** in **Site settings**; on a new site you can pick the `member` role it starts with. Until then, members who sign up or claim a Wikidot account hold only `registered`, which has no permissions by default, so they cannot edit pages. See [Site administration](https://pwikit-docs.unitreaty.org/en/site-administration#registration-and-accounts).
>
> The `member` and `admin` roles a new site starts with are only a generic starting point. Adjust their permissions under **Roles** to match your site's rules soon as well. See [Built-in roles](https://pwikit-docs.unitreaty.org/en/site-administration#built-in-roles).

- From now on `pwikit` works from any directory, for example `pwikit site list`. On Windows, open a new terminal first.
- Sign in at `/-/login` on the site's domain; the admin panel is at `/-/admin`.
- To send verification and password reset mail, fill in the `[mail]` section of `pwikit.toml`. Until then those messages are written to the log instead of being sent. See [Configuration](https://pwikit-docs.unitreaty.org/en/configuration).
- Claim verification codes are sent by a shared service by default. To send them from a Wikidot account of your own, see [Claim verification codes](https://pwikit-docs.unitreaty.org/en/deployment#claim-verification-codes).
- Set the site title, theme, roles and permissions in [Site administration](https://pwikit-docs.unitreaty.org/en/site-administration).
- Take backups with `pwikit backup create`. See [Operations](https://pwikit-docs.unitreaty.org/en/operations).
- Installed as a system service, pwikit updates itself to new releases early in the morning and rolls back when an update fails. See [Updating and rolling back](https://pwikit-docs.unitreaty.org/en/updating).
- To move the instance to another disk or directory later, register the service again as described in [Moving an instance](https://pwikit-docs.unitreaty.org/en/deployment#moving-an-instance).

## If a step stops

| What you see | What to do |
|---|---|
| Typing `pwikit` gives "command not found" | Before step 4, write `./pwikit`. If it is still not found after installing, run `./pwikit path status`; on Windows, open a new terminal first |
| It refuses to run as root | This happens on macOS. Use an ordinary account; the bundled PostgreSQL refuses root |
| It says the bundled PostgreSQL cannot enter a directory | On Linux as root, PostgreSQL runs under its own account. Move pwikit out of `/root` to `/opt` or `/srv` |
| It cannot listen on port 80 or 443 | Install it as a system service, or see [Deployment](https://pwikit-docs.unitreaty.org/en/deployment#cannot-listen-on-port-80-or-443) |
| Every address shows **No such site** | No site is bound to that address. Check the domains given to `createsite`, or change them with `pwikit site rebind` |
| The import stops with `the backup names authors but holds no accounts` | `_users/` was not read, usually because it is not next to the site directory. Nothing was written, so put it in `archive/` and run `./pwikit import` again. See [Importing from Wikidot](https://pwikit-docs.unitreaty.org/en/importing#running-the-import) |
| The import stops with `archive does not exist` | The data directory has no `archive/`. Check that the command runs the pwikit from step 1, and put the backup into its `archive/` |
| The import stops with `no site under "<path>"` | Neither `archive/` nor any directory right below it holds `meta/site.json`. Check the level the backup was put at |
| Pages have no styles | This build does not carry the page assets. See [Deployment](https://pwikit-docs.unitreaty.org/en/deployment#page-assets) |

For other problems, see Troubleshooting in [Deployment](https://pwikit-docs.unitreaty.org/en/deployment#troubleshooting) and, for imports, [Importing from Wikidot](https://pwikit-docs.unitreaty.org/en/importing).
