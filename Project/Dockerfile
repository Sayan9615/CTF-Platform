
FROM ubuntu:22.04


ENV DEBIAN_FRONTEND=noninteractive


RUN apt-get update && \
    apt-get install -y openssh-server sudo zip unzip nano vim tar curl net-tools && \
    apt-get clean


RUN mkdir /var/run/sshd


RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd


RUN usermod -aG sudo student


EXPOSE 22


CMD ["/usr/sbin/sshd", "-D"]
