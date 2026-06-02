# Copyright (c) 2024-2026 MorCherlf
#
# Licensed under the MIT License. You may not use this file except in compliance
# with the License. You may obtain a copy of the License at:
#
#     https://opensource.org/licenses/MIT
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ---- build stage ----
FROM golang:1.25 AS builder

WORKDIR /src

# Cache module downloads. The build context is the repository root so that the
# example can compile against the local SDK module.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static binary so it runs in a distroless runtime.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/deepseek ./examples/deepseek

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

# The platform injects the broker/cache credentials and the plugin identity at
# container start. The plugin-specific keys are populated from the boot config.
#
#   RABBITMQ_HOST, RABBITMQ_PORT, RABBITMQ_USER, RABBITMQ_PASSWORD
#   REDIS_HOST, REDIS_PORT, REDIS_USER, REDIS_PASSWORD
#   GRIFFINO_PLUGIN_ID, GRIFFINO_USER_ID
#   DEEPSEEK_API_KEY, DEEPSEEK_MODEL, DEEPSEEK_BASE_URL, MAX_TOKENS, SYSTEM_PROMPT

COPY --from=builder /out/deepseek /usr/local/bin/deepseek

ENTRYPOINT ["/usr/local/bin/deepseek"]
