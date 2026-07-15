#!/bin/bash


set -e

/usr/sbin/sshd -D &
SSHD_PID=$!

echo "[MONITOR] sshd pornit (PID $SSHD_PID). Aștept prima conexiune..."


while true; do
    ACTIVE=$(ss -tn state established '( dport = :22 or sport = :22 )' 2>/dev/null | tail -n +2 | wc -l)
    if [ "$ACTIVE" -gt 0 ]; then
        echo "[MONITOR] Conexiune detectata. Încep monitorizarea pentru deconectare."
        break
    fi
    sleep 3
done


while true; do
    sleep 5
    ACTIVE=$(ss -tn state established '( dport = :22 or sport = :22 )' 2>/dev/null | tail -n +2 | wc -l)
    if [ "$ACTIVE" -eq 0 ]; then
        echo "[MONITOR] Nicio conexiune activă. Opresc instanța."
        kill -TERM "$SSHD_PID"
        break
    fi
done

wait "$SSHD_PID"