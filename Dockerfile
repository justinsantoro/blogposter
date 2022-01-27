FROM alpine:3.13 AS installer-env

# Define Args for the needed to add the package
ARG PS_VERSION=7.0.0
ARG PS_PACKAGE=powershell-${PS_VERSION}-linux-alpine-x64.tar.gz
ARG PS_PACKAGE_URL=https://github.com/PowerShell/PowerShell/releases/download/v${PS_VERSION}/${PS_PACKAGE}
ARG PS_INSTALL_VERSION=7

# Download the Linux tar.gz and save it
ADD ${PS_PACKAGE_URL} /tmp/linux.tar.gz

# define the folder we will be installing PowerShell to
ENV PS_INSTALL_FOLDER=/opt/microsoft/powershell/$PS_INSTALL_VERSION

# Create the install folder
RUN mkdir -p ${PS_INSTALL_FOLDER}

# Unzip the Linux tar.gz
RUN tar zxf /tmp/linux.tar.gz -C ${PS_INSTALL_FOLDER} -v

FROM klakegg/hugo:0.83.1-ext-pandoc-ci as hugo

# Copy only the files we need from the previous stage
COPY --from=installer-env ["/opt/microsoft/powershell", "/opt/microsoft/powershell"]

COPY . /blogposter
RUN cd /blogposter && go build -o /usr/bin/blogposter
ENV BLOGPOSTER_PORT=80 \
    BLOGPOSTER_PATH=/blogrepo

# Define Args and Env needed to create links
ARG PS_INSTALL_VERSION=7
ENV PS_INSTALL_FOLDER=/opt/microsoft/powershell/$PS_INSTALL_VERSION \
    \
    # Define ENVs for Localization/Globalization
    DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=false \
    LC_ALL=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    # set a fixed location for the Module analysis cache
    PSModuleAnalysisCachePath=/var/cache/microsoft/powershell/PSModuleAnalysisCache/ModuleAnalysisCache \
    POWERSHELL_DISTRIBUTION_CHANNEL=PSDocker-Alpine-3.13


# Install dotnet dependencies and ca-certificates
RUN apk add --no-cache \
    git \
    ca-certificates \
    less \
    \
    # .NET Core dependencies
    krb5-libs \
    libgcc \
    libintl \
    libssl1.1 \
    libstdc++ \
    tzdata \
    userspace-rcu \
    zlib \
    icu-libs \
    && apk -X https://dl-cdn.alpinelinux.org/alpine/edge/main add --no-cache \
    lttng-ust \
    \
    # Create the pwsh symbolic link that points to powershell
    && ln -s ${PS_INSTALL_FOLDER}/pwsh /usr/bin/pwsh \
    \
    # Give all user execute permissions and remove write permissions for others
    && chmod a+x,o-w ${PS_INSTALL_FOLDER}/pwsh \
    # and disable telemetry
    && export POWERSHELL_TELEMETRY_OPTOUT=1

COPY entrypoint.ps1 /

ENTRYPOINT [ "pwsh", "-File", "/entrypoint.ps1" ]