FROM alpine:V0.0.00.002-openshift

RUN groupadd -g 2030 nexagroup && useradd -u 2030 -g nexagroup nexa -d /home/nexa -m

ENV APP_ROOT=/home/nexa
ENV PATH=${APP_ROOT}/bin:${PATH} HOME=${APP_ROOT}

RUN mkdir ${APP_ROOT}/bin

RUN mkdir  ${APP_ROOT}/certs
RUN mkdir  ${APP_ROOT}/conf

# 复制程序和证书文件
COPY server.key server.crt ${APP_ROOT}/certs/

RUN chmod -R +rx ${APP_ROOT} && \
    chown -R nexa:nexagroup ${APP_ROOT}

COPY --chmod=757  nexa ${APP_ROOT}/bin/

RUN chown -R  nexa:nexagroup ${APP_ROOT}/conf

USER 2030
WORKDIR ${APP_ROOT}

ENTRYPOINT ["/home/nexa/bin/nexa"]