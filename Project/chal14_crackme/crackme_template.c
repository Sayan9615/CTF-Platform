#include <stdio.h>
#include <string.h>

int main() {
    char input[128];
    printf("Introdu parola: ");
    if (fgets(input, sizeof(input), stdin) == NULL) {
        return 1;
    }
    input[strcspn(input, "\n")] = 0;

    if (strcmp(input, "FLAG_PLACEHOLDER") == 0) {
        printf("Access Granted! Flag: %s\n", input);
    } else {
        printf("Access Denied.\n");
    }

    return 0;
}
