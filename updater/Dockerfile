FROM docker:cli

RUN apk add --no-cache bash git jq curl docker-cli-compose docker-cli-buildx

COPY updater.sh /usr/local/bin/updater.sh
RUN chmod +x /usr/local/bin/updater.sh

ENTRYPOINT ["/usr/local/bin/updater.sh"]
