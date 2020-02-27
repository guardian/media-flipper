#!/bin/sh

echo Using access key ID \"${AWS_ACCESS_KEY_ID}\"
echo ${TRANSCODED_MEDIA}
ls -lh "${TRANSCODED_MEDIA}"
echo ${THUMBNAIL_IMAGE}
ls -lh "${THUMBNAIL_IMAGE}"

if [ "${FILE_NAME}" != "" ]; then
  #make sure any leading / is removed as s3 does not really like these (well s3 does not technically care but it makes finding stuff a pain)
    OUTPUT_PATH=`dirname "${FILE_NAME}" | sed  's/^\///'`
    echo Using output path ${OUTPUT_PATH} from media file path
else
    echo Using output path ${OUTPUT_PATH} from settings
fi

echo Uploading \"${TRANSCODED_MEDIA}\" to \"${OUTPUT_PATH}\" on \"${OUTPUT_BUCKET}\"...
aws s3 cp "${TRANSCODED_MEDIA}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${MEDIA_FILE}`"

if [ "${THUMBNAIL_IMAGE}" != "" ]; then
  aws s3 cp "${THUMBNAIL_IMAGE}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${THUMBNAIL_IMAGE}`"
fi

exit $?