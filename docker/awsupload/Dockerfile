FROM python:3.8-alpine

RUN pip install awscli
RUN adduser aws -D
COPY awsupload/do_upload.sh /usr/bin
RUN chmod a+x /usr/bin/do_upload.sh
CMD /usr/bin/do_upload.sh
USER aws
