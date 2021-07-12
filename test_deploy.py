from os import system
from time import sleep


system("docker stop rpi-simulator")
sleep(3)
system("docker run -d --rm --name rpi-simulator -p 22:22 rpi-simulator")
system("python main.py")
