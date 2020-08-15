
capture:
	docker-compose exec x11 bash -c "xwd -root -display :0 > capture.xwd && convert capture.xwd capture.jpg"
	docker cp $(shell docker-compose ps -q x11):/capture.jpg capture.jpg
	open capture.jpg