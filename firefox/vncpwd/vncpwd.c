#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <unistd.h>
#include "d3des.h"

static u_char obfKey[8] = {23,82,107,6,35,78,88,7};

void decryptPw( unsigned char *pPW ) {
    unsigned char clrtxt[10];
	
    deskey(obfKey, DE1);
    des(pPW, clrtxt);
    clrtxt[8] = 0;

    fprintf(stdout, "Password: %s\n", clrtxt);
}

void encryptPw(unsigned char * pw, FILE* fp) {
  unsigned char bytes[100] = {0};

  deskey(obfKey, EN0);
  des(pw, bytes);

  fwrite(bytes, sizeof(char) * 8, 1, fp);

  printf("Write binary: ");
  for (int i = 0; i < 8; ++i)
    printf("0x%x ", bytes[i]);
  printf("\n");
}

int main(int argc, char *argv[]) {
    FILE *fp;
    unsigned char *pwd;

    if (argc < 3) {
        fprintf(stdout, "Usage: vncpwd <password file> <mode> [password]\n");
        return 1;
    }

    if (strcmp(argv[2], "get") == 0) {
      if ((fp = fopen(argv[1], "r")) == NULL) {
          fprintf(stderr, "Error: can not open password file: %s\n", argv[1]);
          return 1;
      }
      pwd = malloc(1024);
      fread(pwd, 1024, 1, fp);
      decryptPw(pwd);
      fclose(fp);
      free(pwd);
    }

    if (strcmp(argv[2], "set") == 0) {
      if (argc < 4) {
        fprintf(stdout, "Usage: vncpwd <password file> <mode> [password]\n");
        return 1;
      }
      if ((fp = fopen(argv[1], "w")) == NULL) {
          fprintf(stderr, "Error: can not open password file: %s\n", argv[1]);
          return 1;
      }
      encryptPw((unsigned char *)argv[3], fp);
      fclose(fp);
    }
 
    return 0;
}
