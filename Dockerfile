FROM golang:1.6

ENV GOLANG_VERSION 1.6
ENV GOLANG_SRC_URL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz
ENV GOLANG_SRC_SHA256 a96cce8ce43a9bf9b2a4c7d470bc7ee0cb00410da815980681c8353218dcf146

ENV GOLANG_BOOTSTRAP_VERSION 1.4.3
ENV GOLANG_BOOTSTRAP_URL https://golang.org/dl/go$GOLANG_BOOTSTRAP_VERSION.src.tar.gz
ENV GOLANG_BOOTSTRAP_SHA1 486db10dc571a55c8d795365070f66d343458c48
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN echo "export PATH=$GOPATH/bin:$PATH" >> /etc/bash.bashrc

RUN apt-get update
RUN apt-get -y upgrade

RUN apt-get -y install pkg-config
RUN apt-get -y install libglib2.0-0
RUN apt-get -y install libglib2.0-cil-dev
RUN apt-get -y install libwebkitgtk-3.0-common libwebkitgtk-3.0-dev

RUN apt-get -y install openssh-server
RUN echo "root:root" | chpasswd
RUN /bin/echo -e '#!/bin/sh' > /start.sh
RUN /bin/echo -e '/usr/sbin/sshd' >> /start.sh
RUN /bin/echo -e '/bin/bash' >> /start.sh
RUN chmod 755 /start.sh
RUN sed -i 's/PermitRootLogin without-password/PermitRootLogin yes/' /etc/ssh/sshd_config
RUN mkdir /var/run/sshd


RUN go get github.com/gbyoung/squeegee
RUN go get github.com/gbyoung/squeegee/squeegee

ENTRYPOINT ["/start.sh"]
WORKDIR $GOPATH/src/github.com/gbyoung/squeegee
