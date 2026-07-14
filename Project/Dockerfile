# Folosim Ubuntu ca bază pentru mediul studenților
FROM ubuntu:22.04

# Evităm blocajele care cer "Yes/No" în timpul instalării pachetelor
ENV DEBIAN_FRONTEND=noninteractive

# 1. Instalăm utilitarele esențiale pentru CTF și serverul SSH
RUN apt-get update && \
    apt-get install -y openssh-server sudo zip unzip nano vim tar curl net-tools && \
    apt-get clean

# 2. Configurăm folderul necesar pentru ca SSH să funcționeze
RUN mkdir /var/run/sshd

# 3. Creăm utilizatorul "student" și îi punem parola "student"
# -m (creează home directory), -s (setăm bash ca terminal implicit)
RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd

# (Opțional) Îi dăm drepturi de sudo studentului, dacă vrei provocări de escaladare a privilegiilor
RUN usermod -aG sudo student

# 4. Portul intern pe care va asculta SSH-ul
EXPOSE 22

# 5. Comanda care menține containerul pornit și ascultă conexiuni SSH
CMD ["/usr/sbin/sshd", "-D"]
