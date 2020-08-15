build:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down --remove-orphans
	docker volume rm server-linux_x11socket

vnc:
	open vnc://localhost:5900

capture:
	docker-compose exec x11 bash -c "xwd -root -display :0 > capture.xwd && convert capture.xwd capture.jpg"
	docker cp $(shell docker-compose ps -q x11):/capture.jpg capture.jpg
	open capture.jpg