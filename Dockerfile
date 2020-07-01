FROM scratch

COPY ./build/jx-role-controller /jx-role-controller

CMD ["/jx-role-controller"]
