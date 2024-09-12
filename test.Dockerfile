FROM ubuntu:noble
WORKDIR "/mageanoid"
COPY test.sh test.sh
RUN chmod +x test.sh
ENV XDG_DATA_HOME=/root/.local/share
ENV XDG_STATE_HOME=/root/.local/state
ENV XDG_CACHE_HOME=/root/.cache
ENV XDG_RUNTIME_DIR=/root/.local/run
ENV XDG_CONFIG_HOME=/root/.config
ENTRYPOINT [ "./test.sh" ]
