FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive


RUN apt-get update && apt-get install -y openssh-server iproute2 binutils && apt-get clean
RUN mkdir /var/run/sshd

RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd


COPY imagine.jpg /home/student/imagine.jpg
RUN chown student:student /home/student/imagine.jpg

COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

EXPOSE 22
CMD ["/usr/local/bin/monitor.sh"]