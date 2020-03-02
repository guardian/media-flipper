#!/bin/sh

echo Using access key ID \"${AWS_ACCESS_KEY_ID}\"
echo ${TRANSCODED_MEDIA}
ls -lh "${TRANSCODED_MEDIA}"
echo ${THUMBNAIL_IMAGE}
ls -lh "${THUMBNAIL_IMAGE}"

STRIP_PATH_COUNT=`echo ${CUSTOM_ARGS}| sed -E 's/^.*stripPath=([^,]*).*$/\1/'`
if [ "${STRIP_PATH_COUNT}" == "${CUSTOM_ARGS}" ]; then
  STRIP_PATH_COUNT=""
else
  echo Strip path is ${STRIP_PATH_COUNT}
fi

if [ "${FILE_NAME}" != "" ]; then
  #make sure any leading / is removed as s3 does not really like these (well s3 does not technically care but it makes finding stuff a pain)
    OUTPUT_PATH=`dirname "${FILE_NAME}" | sed  's/^\///'`
    echo Using output path ${OUTPUT_PATH} from media file path
else
    echo Using output path ${OUTPUT_PATH} from settings
fi

if [ "${STRIP_PATH_COUNT}" != "" ]; then
    echo Removing ${STRIP_PATH_COUNT} segments from upload path
    NEW_OUTPUT_PATH=`echo ${OUTPUT_PATH} | sed -E 's/([^\/]+\/){'${STRIP_PATH_COUNT}'}//'`
    if [ "${NEW_OUTPUT_PATH}" == "" ]; then
      echo Removed too many segments!
    else
      OUTPUT_PATH=${NEW_OUTPUT_PATH}
    fi
    echo Output path is now ${OUTPUT_PATH}
fi

FAILED=0
echo Uploading \"${TRANSCODED_MEDIA}\" to \"${OUTPUT_PATH}\" on \"${OUTPUT_BUCKET}\"...
aws s3 cp "${TRANSCODED_MEDIA}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${MEDIA_FILE}`"

if [ "$?" == "" ]; then
  FAILED=$?
fi

if [ "${THUMBNAIL_IMAGE}" != "" ]; then
  aws s3 cp "${THUMBNAIL_IMAGE}" "s3://${OUTPUT_BUCKET}/${OUTPUT_PATH}/`basename ${THUMBNAIL_IMAGE}`"
fi

if [ "$?" == "" ]; then
  FAILED=$?
fi

if [ "$FAILED" == "0" ]; then
  echo Removing local transcode copy...
  rm -f "${TRANSCODED_MEDIA}"
  if [ "${THUMBNAIL_IMAGE}" != "" ]; then
    rm -f "${THUMBNAIL_IMAGE}"
  fi
fi

exit $FAILED