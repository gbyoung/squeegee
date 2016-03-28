FROM ubuntu
MAINTAINER Gary Young <gary.b.young@gmail.com>

# Set environment vars
ENV GO_VERSION go1.6
ENV EXTENSION linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin
ENV GOSRC $GOPATH/src
ENV APP app

# Install go deps, xvfb (x session), libwebkit, gtk, and gotk3
RUN apt-get update -y \
  && apt-get install --no-install-recommends -yq \
    wget \
    build-essential \
    ca-certificates \
    git \
    mercurial \
    bzr \
    dbus \
    xvfb \
    libwebkit2gtk-3.0-dev \
    libgtk-3-dev \
    libcairo2-dev \
  && wget https://storage.googleapis.com/golang/${GO_VERSION}.${EXTENSION} -o /tmp/${GO_VERSION}.${EXTENSION} \
  && tar -zxvf ${GO_VERSION}.${EXTENSION} -C /usr/local \
  && rm ${GO_VERSION}.${EXTENSION} \
  && mkdir $HOME/go \
  && go get -u -tags gtk_3_10 github.com/pasangsherpa/webloop/...

RUN apt-get -y install openssh-server
RUN echo "root:root" | chpasswd
RUN /bin/echo -e '#!/bin/sh' > /start.sh
RUN /bin/echo -e '/usr/sbin/sshd' >> /start.sh
RUN /bin/echo -e 'set -e' >> /start.sh

RUN /bin/echo -e 'DISPLAY=:1' >> /start.sh 
RUN /bin/echo -e 'XVFB=/usr/bin/Xvfb' >> /start.sh
RUN /bin/echo -e 'XVFBARGS="$DISPLAY -ac -screen 0 1024x768x16 +extension RANDR"' >> /start.sh
RUN /bin/echo -e 'PIDFILE="/var/xvfb.pid"' >> /start.sh
RUN /bin/echo -e 'export DISPLAY=$DISPLAY' >> /start.sh
RUN /bin/echo -e '/sbin/start-stop-daemon --start --quiet --pidfile $PIDFILE --make-pidfile --background --exec $XVFB -- $XVFBARGS' >> /start.sh
RUN /bin/echo -e 'sleep 1' >> /start.sh
RUN /bin/echo -e '/bin/bash' >> /start.sh
RUN chmod 755 /start.sh
RUN sed -i 's/PermitRootLogin without-password/PermitRootLogin yes/' /etc/ssh/sshd_config
RUN mkdir /var/run/sshd

RUN go get github.com/gbyoung/squeegee
RUN go get github.com/gbyoung/squeegee/squeegee

ENTRYPOINT ["/start.sh"]
WORKDIR $GOPATH/src/github.com/gbyoung/squeegee
