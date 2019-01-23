ARG ALPINE_BASE
FROM ${ALPINE_BASE} as builder

RUN echo "Would be building"

FROM ${ALPINE_BASE} as unit-test

RUN echo "would be doing unit test and lint stuff here"

FROM ${ALPINE_BASE} as controller

RUN echo "Would be copying build results from builder"
