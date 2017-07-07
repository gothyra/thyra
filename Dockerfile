FROM centos:7

COPY thyra bin/thyra
COPY static static

EXPOSE 3030
CMD [ "/bin/thyra" ]
