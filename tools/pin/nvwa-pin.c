#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <sys/file.h>
#include <sys/ioctl.h>


#define PIN_MEM_MAGIC 0x59
#define _SET_PIN_MEM_AREA    1
#define _CLEAR_PIN_MEM_AREA  2
#define _REMAP_PIN_MEM_AREA  3
#define _FINISH_PIN_MEM_DUMP 4
#define _INIT_PAGEMAP_READ   5
#define _IOC_MAX_NR          5
#define CLEAR_PIN_MEM_AREA      _IOW(PIN_MEM_MAGIC, _CLEAR_PIN_MEM_AREA, int)
#define REMAP_PIN_MEM_AREA      _IOW(PIN_MEM_MAGIC, _REMAP_PIN_MEM_AREA, int)
#define FINISH_PIN_MEM_DUMP     _IOW(PIN_MEM_MAGIC, _FINISH_PIN_MEM_DUMP, int)
#define INIT_PAGEMAP_READ       _IOW(PIN_MEM_MAGIC, _INIT_PAGEMAP_READ, int)
#define PIN_MEM_FILE  "/dev/pinmem"
#define INPUT_PARA_NUM  1

int main(int argc , char *argv[])
{
	int fd;
	int ret = 0;
	int para = 0;

	fd = open(PIN_MEM_FILE,O_RDWR);
	if (fd <= 0) {
		printf("Open file:%s fail.\n", PIN_MEM_FILE);
		return -1;
	}
	if (argc < INPUT_PARA_NUM + 1) {
		close(fd);
		return -EINVAL;
	}
	if (!strcmp(argv[1], "--finish-pin")) {
		ret = ioctl(fd, FINISH_PIN_MEM_DUMP, &para);
		if (ret < 0) {
			printf("Finish pin fail, errno: %s\n", strerror(errno));
		}
	} else if (!strcmp(argv[1], "--clear-pin-mem")) {
		ret = ioctl(fd, CLEAR_PIN_MEM_AREA, &para);
		if (ret < 0) {
			printf("Clear pin mem fail, errno: %s\n", strerror(errno));
		}
	} else if (!strcmp(argv[1], "--init-pagemap-read")) {
		ret = ioctl(fd, INIT_PAGEMAP_READ, &para);
		if (ret < 0) {
			printf("Init pagemap read fail, errno: %s\n", strerror(errno));
		}
	}

	close(fd);
	return ret;
}
