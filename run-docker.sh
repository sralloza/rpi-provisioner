docker build -t rpi-simulator .
docker run -d --rm --name rpi-simulator -p 22:22 rpi-simulator
docker stop rpi-simulator && docker run -d --rm --name rpi-simulator -p 22:22 rpi-simulator
