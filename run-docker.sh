docker build -t rpi-simulator .
docker run -d --rm -p 22:22 rpi-simulator
