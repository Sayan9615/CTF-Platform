FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y openssh-server iproute2 sudo && apt-get clean
RUN mkdir /var/run/sshd

RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd

RUN touch /root/secret.txt && chmod 600 /root/secret.txt

RUN echo 'student ALL=(root) NOPASSWD: /usr/bin/cat /root/secret.txt' > /etc/sudoers.d/student_cat && \
    chmod 440 /etc/sudoers.d/student_cat

COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

EXPOSE 22
CMD ["/usr/local/bin/monitor.sh"]