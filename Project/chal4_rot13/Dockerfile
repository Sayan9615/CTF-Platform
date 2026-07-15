FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y openssh-server iproute2 && apt-get clean
RUN mkdir /var/run/sshd

RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd

# mesaj_secret.txt e complet generat la runtime de server.go
# (conținut ROT13, injectat prin buildInjectCommand, case 4)
RUN touch /home/student/mesaj_secret.txt && chown student:student /home/student/mesaj_secret.txt

COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

EXPOSE 22
CMD ["/usr/local/bin/monitor.sh"]