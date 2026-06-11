#include <cstdio>
#include <cstdlib>
#include <cstring>
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

	sockaddr_in addr{};
	addr.sin_family = AF_INET;
	addr.sin_port = htons(8080);
	addr.sin_addr.s_addr = htonl(INADDR_ANY);

	if (bind(fd, reinterpret_cast<sockaddr *>(&addr), sizeof(addr)) < 0) {
		perror("bind");
		return 1;
	}
	if (listen(fd, 5) < 0) {
		perror("listen");
		return 1;
	}

	const char resp[] =
		"HTTP/1.0 200 OK\r\n"
		"Content-Type: text/plain\r\n"
		"Content-Length: 19\r\n"
		"Connection: close\r\n"
		"\r\n"
		"hello from C++\n";

	for (;;) {
		int cl = accept(fd, nullptr, nullptr);
		if (cl < 0) {
			perror("accept");
			continue;
		}
		write(cl, resp, std::strlen(resp));
		close(cl);
	}
}
