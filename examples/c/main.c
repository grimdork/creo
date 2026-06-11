#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <netinet/in.h>

int main() {
	int fd = socket(AF_INET, SOCK_STREAM, 0);
	if (fd < 0) {
		perror("socket");
		return 1;
	}

	int opt = 1;
	setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

	struct sockaddr_in addr = {
		.sin_family = AF_INET,
		.sin_port = htons(8080),
		.sin_addr = { htonl(INADDR_ANY) },
	};
	if (bind(fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
		perror("bind");
		return 1;
	}
	if (listen(fd, 5) < 0) {
		perror("listen");
		return 1;
	}

	char resp[] =
		"HTTP/1.0 200 OK\r\n"
		"Content-Type: text/plain\r\n"
		"Content-Length: 13\r\n"
		"Connection: close\r\n"
		"\r\n"
		"hello from C\n";

	for (;;) {
		int cl = accept(fd, NULL, NULL);
		if (cl < 0) {
			perror("accept");
			continue;
		}
		write(cl, resp, strlen(resp));
		close(cl);
	}
}
