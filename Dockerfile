FROM intelsdi/snap:alpine
RUN apk update && apk add curl
RUN curl -sLO http://hyperpilot-snap-collectors.s3.amazonaws.com/snap-plugin-collector-docker
RUN curl -sLO https://github.com/intelsdi-x/snap-plugin-collector-cpu/releases/download/6/snap-plugin-collector-cpu_linux_x86_64
RUN curl -sLO https://github.com/intelsdi-x/snap-plugin-collector-meminfo/releases/download/4/snap-plugin-collector-meminfo_linux_x86_64
RUN curl -sLO https://github.com/intelsdi-x/snap-plugin-collector-disk/releases/download/5/snap-plugin-collector-disk_linux_x86_64
RUN curl -sLO https://github.com/intelsdi-x/snap-plugin-publisher-file/releases/download/2/snap-plugin-publisher-file_linux_x86_64
RUN curl -sLO https://github.com/intelsdi-x/snap-plugin-publisher-influxdb/releases/download/22/snap-plugin-publisher-influxdb_linux_x86_64
RUN cp /opt/snap/bin/snaptel /bin/snaptel
RUN chmod +x /bin/snaptel
COPY bin/snap-average-counter-processor /
COPY test-task.yaml /
CMD /usr/local/bin/init_snap && /opt/snap/sbin/snapteld -t ${SNAP_TRUST_LEVEL} -l ${SNAP_LOG_LEVEL} -o '/' & \
    sleep 1; \
    snaptel plugin load /snap-plugin-publisher-influxdb_linux_x86_64; \
    snaptel plugin load /snap-average-counter-processor; \
    snaptel plugin load /snap-plugin-collector-docker; \
    snaptel plugin load /snap-plugin-collector-cpu_linux_x86_64; \
    snaptel plugin load /snap-plugin-collector-meminfo_linux_x86_64; \
    snaptel plugin load /snap-plugin-collector-disk_linux_x86_64; \
    snaptel plugin load /snap-plugin-publisher-file_linux_x86_64; \
    snaptel task create -t /test-task.yaml; \
    sleep 10; \
    tail -f /tmp/log/processor.log
