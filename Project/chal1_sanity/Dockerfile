# 1. Imaginea de bază: Ubuntu curat
FROM ubuntu:22.04

# 2. Setări pentru a evita blocajele în timpul instalării
ENV DEBIAN_FRONTEND=noninteractive

# 3. Instalăm serverul SSH + iproute2 (pentru comanda 'ss', necesară
#    scriptului de monitorizare a conexiunilor active)
RUN apt-get update && apt-get install -y openssh-server iproute2 && apt-get clean
RUN mkdir /var/run/sshd

# 4. Creăm contul studentului (username: student, parola: student)
RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd

# 5. Pregătim fișierul pentru flag
# Inițial punem un text standard. Serverul Go va înlocui acest text cu flag-ul unic generat
# exact în momentul în care studentul pornește instanța!
RUN echo "Acesta este un fișier temporar. Dacă vezi asta, Go nu a injectat flag-ul." > /home/student/bun_venit.txt

# 6. Setăm permisiunile (studentul poate doar să citească, nu să modifice sau să șteargă fișierul)
RUN chown root:root /home/student/bun_venit.txt
RUN chmod 644 /home/student/bun_venit.txt

# 7. Copiem scriptul de monitorizare care oprește containerul automat
#    când studentul se deconectează (exit din SSH)
COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

# 8. Expunem portul pentru SSH
EXPOSE 22

# 9. Pornim scriptul de monitorizare în loc de sshd direct.
#    monitor.sh pornește sshd intern și oprește containerul
#    când nu mai există nicio conexiune activă.
CMD ["/usr/local/bin/monitor.sh"]