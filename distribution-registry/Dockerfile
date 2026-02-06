FROM docker.io/library/registry

COPY start.sh /start.sh
COPY config.yml /etc/distribution/config.yml

RUN chmod +x /start.sh

ENTRYPOINT ["/start.sh"]

EXPOSE 5000

ENV OTEL_TRACES_EXPORTER="none"