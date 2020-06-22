FROM centos:7

COPY ./build/jx-role-controller-linux-amd64 /jx-role-controller

CMD ["/jx-role-controller"]
