#!/bin/sh

echo Using access key ID \"${AWS_ACCESS_KEY_ID}\"
echo ${TRANSCODED_MEDIA}
ls -lh "${TRANSCODED_MEDIA}"
echo ${THUMBNAIL_IMAGE}
ls -lh "${THUMBNAIL_IMAGE}"
echo Uploading \"${TRANSCODED_MEDIA}\" to \"${OUTPUT_PATH}\" on \"${OUTPUT_BUCKET}\"...
aws s3 cp "${TRANSCODED_MEDIA}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${MEDIA_FILE}`"

if [ "${THUMBNAIL_IMAGE}" != "" ]; then
  aws s3 cp "${THUMBNAIL_IMAGE}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${THUMBNAIL_IMAGE}`"
fi

exit $?