FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive


RUN apt-get update && apt-get install -y \
    openssh-server iproute2 curl build-essential libssl-dev zlib1g-dev \
    yasm libgmp-dev libpcap-dev pkg-config libbz2-dev && \
    apt-get clean

RUN mkdir /var/run/sshd


RUN curl -L https://github.com/openwall/john/archive/refs/heads/bleeding-jumbo.tar.gz -o john.tar.gz && \
    tar -xzf john.tar.gz && \
    cd john-bleeding-jumbo/src && \
    ./configure --disable-openmp && make -s clean && make -sj4 && \
    mkdir -p /opt/john && cp -a ../run/* /opt/john/ && \
    echo '#!/bin/bash\nexec /opt/john/john "$@"' > /usr/local/bin/john && \
    chmod +x /usr/local/bin/john && \
    cd / && rm -rf john.tar.gz john-bleeding-jumbo


RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd


RUN chown -R student:student /opt/john


RUN touch /home/student/hash.txt /home/student/wordlist.txt && \
    chown student:student /home/student/hash.txt /home/student/wordlist.txt

COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

EXPOSE 22
CMD ["/usr/local/bin/monitor.sh"]