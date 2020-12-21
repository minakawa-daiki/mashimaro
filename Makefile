up:
	docker-compose up -d

down:
	docker-compose down
	docker volume rm mashimaro_x11socket

bash:
	docker-compose exec streamer bash

