# ============================================================
# Dockerfile pentru Challenge 2: Cutia Pandorei
# Build: docker build -f Dockerfile.chal2 -t os-ctf-chal2 .
# (asigură-te că monitor.sh e în același folder ca acest Dockerfile)
# ============================================================

FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y openssh-server iproute2 zip unzip && apt-get clean
RUN mkdir /var/run/sshd

RUN useradd -m -s /bin/bash student
RUN echo 'student:student' | chpasswd

# ------------------------------------------------------------
# Fișiere "decoy" (momeală) - țin loc de fișiere reale de sistem.
# Flag-ul va fi injectat în access.log, printre linii false de log,
# la runtime (vezi server.go / buildInjectCommand, case 2).
# Directorul e în /opt, nu în /home/student, ca studentul să nu
# poată vedea sursele înainte ca arhiva să fie generată.
# ------------------------------------------------------------
RUN mkdir -p /opt/pandora_src

RUN cat > /opt/pandora_src/readme.txt << 'EOF'
Backup automat - sistem de fisiere
-----------------------------------
Acest pachet contine o copie de rezerva partiala a configuratiei
serverului, generata automat de scriptul de mentenanta saptamanala.

Daca ai ajuns aici cautand ceva anume, s-ar putea sa fie nevoie
sa te uiti mai atent prin toate fisierele.
EOF

RUN cat > /opt/pandora_src/notes.txt << 'EOF'
TODO:
- verificat cron job-urile de backup
- actualizat certificatele SSL saptamana viitoare
- discutat cu echipa despre migrarea bazei de date
- nu uita de intalnirea de luni
EOF

RUN cat > /opt/pandora_src/config.yml << 'EOF'
server:
  host: 0.0.0.0
  port: 8080
  timeout: 30s
database:
  driver: postgres
  pool_size: 10
logging:
  level: info
  retention_days: 14
EOF

RUN cat > /opt/pandora_src/access.log << 'EOF'
2026-07-10 09:12:01 INFO  request GET /api/health 200
2026-07-10 09:12:04 INFO  request GET /api/status 200
2026-07-10 09:13:22 WARN  slow query detected (412ms)
2026-07-10 09:14:01 INFO  request POST /api/login 200
2026-07-10 09:15:47 ERROR connection reset by peer
2026-07-10 09:16:03 INFO  request GET /api/health 200
2026-07-10 09:17:19 INFO  cache invalidated for key user:1044
2026-07-10 09:18:02 INFO  request GET /api/health 200
2026-07-10 09:18:45 DEBUG session refreshed for user 882
2026-07-10 09:19:30 INFO  request GET /api/health 200
2026-07-10 09:20:11 WARN  retry attempt 2/3 for job sync-451
2026-07-10 09:21:00 INFO  request GET /api/health 200
2026-07-10 09:22:14 INFO  backup job completed in 4.2s
2026-07-10 09:23:01 INFO  request GET /api/health 200
EOF


RUN echo "FLAG_PLACEHOLDER" > /opt/pandora_src/.sys_cache.dat

RUN cat > /opt/pandora_src/todo.md << 'EOF'
# Sarcini restante

- [ ] Rotire chei API
- [ ] Curatare loguri vechi
- [ ] Verificare integritate backup
- [ ] Update dependinte
EOF

COPY monitor.sh /usr/local/bin/monitor.sh
RUN chmod +x /usr/local/bin/monitor.sh

EXPOSE 22
CMD ["/usr/local/bin/monitor.sh"]