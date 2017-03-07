FROM alpine

RUN mkdir -p /run/docker/plugins

COPY docker-macvlan /usr/bin/docker-macvlan

CMD ["/usr/bin/docker-macvlan"]
