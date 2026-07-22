#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <time.h>

const char *questions[4] = {
    "Care este cea mai apropiata planeta de Soare?",
    "Cum se numeste galaxia noastra?",
    "Ce gaz domina atmosfera lui Marte?",
    "Cati sateliti naturali are Pamantul?"
};

int main() {
    
    srand(time(NULL));

    unsigned char enc_flag[] = { ENC_FLAG_PLACEHOLDER };
    int enc_len = ENC_LEN_PLACEHOLDER;
    unsigned char key = 0x5A;

    char code[11];
    char line[256];

    printf("=== Chestionar rapid ===\n");
    printf("Raspunde sincer la urmatoarele intrebari.\n\n");

    for (int round = 0; round < 10; round++) {
        int qidx = rand() % 4;
        printf("Intrebarea %d: %s\n> ", round + 1, questions[qidx]);
        fflush(stdout);

        if (fgets(line, sizeof(line), stdin) == NULL) {
            line[0] = '\0';
        }

        
        int numar_introdus = 0;
        sscanf(line, "%d", &numar_introdus);
        
        
        code[round] = '0' + (numar_introdus % 10);

        printf("Raspuns inregistrat.\n\n");
    }
    code[10] = '\0'; 

    printf("Acum introdu codul de 10 cifre: ");
    fflush(stdout);

    char input_code[64];
    if (fgets(input_code, sizeof(input_code), stdin) == NULL) {
        input_code[0] = '\0';
    }
    input_code[strcspn(input_code, "\n")] = 0; 

    if (strcmp(input_code, code) == 0) {
        char decoded[256];
        for (int i = 0; i < enc_len; i++) {
            decoded[i] = enc_flag[i] ^ key;
        }
        decoded[enc_len] = '\0';
        printf("Acces permis! Flag: %s\n", decoded);
    } else {
        printf("Cod incorect. Incearca din nou.\n");
    }

    return 0;
}