from os import system
from time import sleep

import pendulum as pm
from beepy import beep
from tabulate import tabulate

t0 = pm.now()
system("docker stop rpi-simulator")
t1 = pm.now()
sleep(3)
t2 = pm.now()
system("docker run -d --rm --name rpi-simulator -p 22:22 rpi-simulator")
t3 = pm.now()
# system("python main.py")
t4 = pm.now()


extra = t2 - t1
total = t4 - t0 - extra
stopped_in = t1 - t0
started_in = t3 - t2
tests_run_in = t4 - t3

data = [
    ["Stop container", stopped_in.in_words()],
    ["Start container", started_in.in_words()],
    ["Test", tests_run_in.in_words()],
    ["Total", total.in_words()],
]
print("\n\n")
print(tabulate(data, headers=["Concept", "Time"]))
# beep(5)
# beep(5)
