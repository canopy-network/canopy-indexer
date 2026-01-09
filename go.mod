module github.com/canopy-network/canopy-indexer

go 1.24.0

require (
	github.com/ThreeDotsLabs/watermill v1.4.4
	github.com/ThreeDotsLabs/watermill-redisstream v1.4.2
	github.com/canopy-network/canopy v0.1.15-0.20251225182851-bcc795889f23
	github.com/canopy-network/canopyx v0.0.0
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	github.com/jackc/pgx/v5 v5.8.0
	github.com/redis/go-redis/v9 v9.14.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.18.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/RaduBerinde/axisds v0.0.0-20250419182453-5135a0650657 // indirect
	github.com/RaduBerinde/btreemap v0.0.0-20250419232817-bf0d809ae648 // indirect
	github.com/Rican7/retry v0.3.1 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/allegro/bigcache/v3 v3.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/crlib v0.0.0-20251024155502-a2e0a212ef05 // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble/v2 v2.1.1 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/cockroachdb/swiss v0.0.0-20250624142022-d6e517c1d961 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/consensys/bavard v0.1.27 // indirect
	github.com/consensys/gnark-crypto v0.16.0 // indirect
	github.com/crate-crypto/go-eth-kzg v1.3.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/drand/kyber v1.3.0 // indirect
	github.com/drand/kyber-bls12381 v0.3.1 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.0 // indirect
	github.com/ethereum/go-ethereum v1.15.11 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/getsentry/sentry-go v0.36.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kilic/bls12-381 v0.1.0 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/minlz v1.0.1 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasisprotocol/curve25519-voi v0.0.0-20230904125328-1f23a7beb09a // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.1 // indirect
	github.com/prometheus/procfs v0.19.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/supranational/blst v0.3.14 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace github.com/canopy-network/canopy => ../canopy/blob

replace github.com/canopy-network/canopyx => ../canopyx
