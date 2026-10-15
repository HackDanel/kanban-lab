# Деплой

## Текущий стенд

| Окружение | Адрес | Ветка |
|---|---|---|
| staging | https://staging.37.194.184.235.sslip.io | `develop` (и `main` перед production) |
| production | https://kanban.37.194.184.235.sslip.io | `main` |

Сервер — домашний `b450` (Debian 13), на нём уже работают другие сервисы за Apache. Поэтому:

- стеки лежат в `~/kanban` пользователя `hackdanel` (`KANBAN_DIR=/home/hackdanel/kanban`), а не в `/opt/kanban`;
- TLS завершает уже установленный Apache: `deploy/apache/setup-https.sh` добавляет два виртуальных хоста по именам `sslip.io` и получает для них сертификаты certbot, не трогая остальные сайты;
- web-контейнеры слушают `127.0.0.1:8101` и `127.0.0.1:8102`, Grafana — `127.0.0.1:3101` и `3102` (доступна по `/grafana/` на том же домене);
- ключ CI в `~/.ssh/authorized_keys` ограничен принудительной командой `deploy/ci-command.sh`: CI передаёт в stdin свой временный `GITHUB_TOKEN`, и сервер этим токеном скачивает файлы деплоя ровно указанного коммита и входит в GHCR, затем выходит. Шелла по ключу не получить, а без действующего токена репозитория нельзя подменить и файлы деплоя;
- репозиторий приватный; образы в GHCR сейчас публичные, но деплой работает и с приватными;
- бэкап баз — cron пользователя, 03:30 ежедневно, файлы в `~/kanban/backups`;
- нагрузочный workflow `load.yml` запускается только вручную и по умолчанию с малой нагрузкой: сервер общий.

Пароли администратора, Grafana и базы каждого окружения сгенерированы на сервере и лежат в `~/kanban/.env.staging` и `~/kanban/.env.production` (права 600).

## Подготовка другого сервера

Ниже — общий порядок для чистого VPS с nginx на хосте.

Staging и production работают на одном VPS как два независимых стека Docker Compose (`kanban-staging`, `kanban-production`) со своими базами и томами. Nginx на хосте принимает HTTPS и проксирует запросы в контейнер `web` нужного стека. Выкладку выполняет GitHub Actions по SSH: `deploy/deploy.sh` скачивает образы с тегом коммита, перезапускает стек и при неуспешной проверке здоровья возвращает предыдущую версию.

## Подготовка сервера (один раз)

Нужен сервер с Ubuntu 24.04 или Debian 12, 2 ГБ памяти и больше, два доменных имени, указывающих на его адрес (например, `staging.example.ru` и `kanban.example.ru`). Без своего домена подойдут имена вида `staging.<ip>.sslip.io` и `<ip>.sslip.io`.

```sh
# Docker и nginx
curl -fsSL https://get.docker.com | sh
apt install -y nginx certbot python3-certbot-nginx

# пользователь для деплоя
adduser --disabled-password deploy
usermod -aG docker deploy
mkdir -p /opt/kanban && chown deploy:deploy /opt/kanban

# ключ, которым GitHub Actions заходит на сервер
sudo -u deploy ssh-keygen -t ed25519 -N '' -f /home/deploy/.ssh/deploy
sudo -u deploy sh -c 'cat ~/.ssh/deploy.pub >> ~/.ssh/authorized_keys'
cat /home/deploy/.ssh/deploy        # закрытый ключ → секрет SSH_KEY
```

Переменные окружений — по образцу `deploy/compose/env.example`:

```sh
sudo -u deploy cp env.example /opt/kanban/.env.staging      # WEB_PORT=8101, GRAFANA_PORT=3101
sudo -u deploy cp env.example /opt/kanban/.env.production   # WEB_PORT=8102, GRAFANA_PORT=3102
chmod 600 /opt/kanban/.env.*
# заполнить пароли и JWT_SECRET (openssl rand -base64 48), разные для двух окружений
```

Nginx и сертификаты:

```sh
sed -e s/STAGING_DOMAIN/staging.example.ru/ -e s/PRODUCTION_DOMAIN/kanban.example.ru/ \
  deploy/nginx/kanban.conf.template > /etc/nginx/sites-available/kanban.conf
ln -s /etc/nginx/sites-available/kanban.conf /etc/nginx/sites-enabled/
certbot --nginx -d staging.example.ru -d kanban.example.ru
systemctl reload nginx
```

Ежедневный бэкап баз:

```sh
echo '30 3 * * * deploy /opt/kanban/backup.sh' > /etc/cron.d/kanban-backup
```

## Настройка репозитория

Settings → Secrets and variables → Actions:

| Тип | Имя | Значение |
|---|---|---|
| secret | `SSH_HOST` | адрес сервера |
| secret | `SSH_PORT` | порт SSH |
| secret | `SSH_USER` | `deploy` |
| secret | `SSH_KEY` | закрытый ключ из шага выше |
| secret | `STAGING_ADMIN_EMAIL`, `STAGING_ADMIN_PASSWORD` | администратор staging (для smoke- и нагрузочных тестов) |
| secret | `CODECOV_TOKEN` | токен Codecov (необязательно) |
| variable | `STAGING_URL` | `https://staging.example.ru` |
| variable | `PRODUCTION_URL` | `https://kanban.example.ru` |
| variable | `DEPLOY_ENABLED` | `true` — включает деплой и нагрузочный workflow |

Окружения `staging` и `production` (Settings → Environments) создаются автоматически при первом деплое; для production можно включить ручное подтверждение.

Ключ деплоя добавляется в `authorized_keys` с принудительной командой:

```
command="KANBAN_DIR=/opt/kanban /opt/kanban/ci-command.sh",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty ssh-ed25519 AAAA… kanban-lab-ci-deploy
```

## Надёжность

- контейнеры перезапускаются при сбое (`restart: unless-stopped`), Docker поднимает их после перезагрузки сервера;
- у backend и web есть проверки здоровья; `deploy.sh` ждёт их и откатывает версию, если новая не поднялась за 2 минуты;
- миграции применяет сервер при старте под advisory-блокировкой PostgreSQL;
- сервер завершает запросы при остановке (graceful shutdown, 15 с);
- PostgreSQL выгружается ежедневно, копии хранятся 14 дней.

## Ручной деплой и откат

```sh
ssh deploy@server sh /opt/kanban/deploy.sh staging <sha-коммита>
cat /opt/kanban/.deployed-staging        # текущая версия
```
