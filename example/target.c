#include <unistd.h>
#include <stdio.h>
#include <string.h>

void boop()
{
    printf("boop");
}

int main(int argc, char **argv)
{
    char buf[16];
    int a = 1;
    int *b = NULL;


    int ret = read(0, buf, 16);
    buf[ret+1] = 0x0;

    if (strstr(buf, "s") == buf)
    {
        if (strstr(buf, "sl") == buf) {
            sleep(5);
        }
    }
    else if (strstr(buf, "b") == buf)
    {
        if (strstr(buf, "bo") == buf) {
            boop();
        }
    }
    else if (strstr(buf, "c") == buf)
    {
        if (strstr(buf, "cr") == buf)
        {
            *b = a;
        }
    }

    return a;
}
