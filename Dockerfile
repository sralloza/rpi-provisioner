FROM debian:latest

RUN apt-get update -y && \
    apt-get upgrade -y && \
    apt-get install sudo -y && \
    apt-get install openssh-server -y && \
    rm -rf /var/lib/apt/lists/*

RUN echo 'root:password' | chpasswd

RUN mkdir /var/run/sshd

RUN sed 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' -i /etc/ssh/sshd_config

RUN cat /etc/ssh/sshd_config


RUN groupadd pi && \
    mv /etc/sudoers /etc/sudoers-backup && \
    (cat /etc/sudoers-backup; echo "%pi ALL=(ALL) ALL") > /etc/sudoers && \
    chmod 440 /etc/sudoers

RUN adduser --gecos 'PI' --disabled-password --ingroup pi pi

RUN echo pi:raspberry | chpasswd

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]
