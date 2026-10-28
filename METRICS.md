# Metrics

jmark의 1 op는 전체 입력에 대해 `run(input)`을 한 번 호출하는 작업이다.

시간과 누적 counter는 반복 횟수를 달리한 두 실행을 비교한다. `W`는 baseline 반복 횟수,
`N`은 추가 반복 횟수다. 각 실행을 여러 번 측정한 뒤 다음 값을 구한다.

`ΔX = median(X(W + N)) - median(X(W))`

## Wall time (`wall_time`)

작업이 끝날 때까지 흐른 시간이다. CPU 실행 시간과 대기 시간을 모두 포함한다.

측정: `Δwall / N`. 저장 단위는 `s/op`이며 화면에는 `µs/op`로 표시한다. 값이 작을수록 빨리 끝난다.

## CPU time (`cpu_time`)

프로세스의 스레드들이 CPU를 사용한 시간을 합한 값이다. 여러 코어를 동시에 사용하면 wall time보다 길어질 수 있다.

측정: `Δcpu / N`. 각 실행의 user time과 system time을 더한 뒤 중앙값 차이를 계산한다. 단위는 `s/op`다.

## User CPU time (`user_time`)

CPU가 애플리케이션과 라이브러리 코드를 실행한 시간이다.

측정: `Δuser / N`. 단위는 `s/op`다.

## System CPU time (`system_time`)

CPU가 프로세스를 위해 커널 코드를 실행한 시간이다. 파일 읽기 같은 시스템 작업이 여기에 포함된다.

측정: `Δsystem / N`. 단위는 `s/op`다.

## Peak RSS (`peak_rss`)

실행 중 물리 메모리 점유가 가장 컸던 순간의 값이다. 런타임 시작과 입력 생성에 사용한 메모리도 포함한다.

측정: `median(peak RSS(W + N))`. 저장 단위는 `B`, 화면 단위는 `MiB`다. baseline을 빼거나 반복 횟수로 나누지 않는다. 할당한 메모리의 총량과는 다르다.

## Throughput (`throughput`)

초당 완료할 수 있는 작업 수다. 값이 클수록 같은 시간에 더 많은 작업을 처리한다.

측정: `1 / wall_time`. 단위는 `op/s`다. 현재 jmark에서는 wall time/op의 역수로 계산한다. 입력 크기 `n`을 초당 항목 수로 해석하지 않는다.

## Instructions (`instructions`)

CPU가 완료한 명령어 수다. 같은 작업을 처리하는 데 얼마나 많은 명령어를 실행했는지 비교한다.

측정: `Δinstructions / N`. 단위는 `instructions/op`다. CPU와 운영체제, 계측 권한에 따라 수집 가능 여부가 달라진다.

## CPU cycles (`cpu_cycles`)

작업을 실행하는 동안 소비한 CPU cycle 수다.

측정: `Δcycles / N`. 단위는 `cycles/op`다. CPU 주파수와 실행 환경이 다르면 시간과의 관계도 달라진다.

## IPC (`ipc`)

CPU cycle 하나당 완료한 명령어 수다. instructions와 cycles의 관계를 보여준다.

측정: `Δinstructions / Δcycles`. 단위는 `instructions/cycle`이다. 서로 다른 코드에서 IPC가 높다는 이유만으로 더 빠르다고 판단하지 않는다.

## Cache references (`cache_references`)

CPU의 generic cache event가 센 캐시 접근 횟수다. 어떤 접근을 세는지는 CPU에 따라 달라진다.

측정: `Δcache_references / N`. 단위는 `references/op`다. 접근 횟수 자체에는 선호 방향을 두지 않는다.

## Cache misses (`cache_misses`)

CPU의 generic cache event가 센 캐시 miss 횟수다.

측정: `Δcache_misses / N`. 단위는 `misses/op`다. CPU마다 event 정의가 다를 수 있으므로 같은 환경에서 비교한다.

## Cache miss rate (`cache_miss_rate`)

수집한 캐시 접근 중 miss가 차지하는 비율이다.

측정: `Δcache_misses / Δcache_references`. 저장 단위는 `ratio`, 화면 단위는 `%`다. 두 counter가 모두 유효하고 접근 횟수가 양수일 때 계산한다.

## Branch misses (`branch_misses`)

CPU가 분기 방향을 잘못 예측한 횟수다.

측정: `Δbranch_misses / N`. 단위는 `misses/op`다. 전체 분기 횟수에 대한 비율은 계산하지 않는다.

## 결과를 읽을 때

시간 차이가 0 이하이거나 counter 차이가 음수면 측정 해상도가 부족한 값으로 처리한다.
지원하지 않는 지표와 수집하지 못한 값은 `unavailable`로 남긴다. 유효한 counter 값 0은 그대로 표시한다.

원시 샘플과 통계는 프로세스 실행 단위다. 개별 op의 p95/p99 지연시간을 뜻하지 않는다.
Peak RSS는 기다린 자식 프로세스를 포함해 가장 큰 프로세스 peak를 나타낸다.
프로세스 트리가 동시에 사용한 메모리의 합은 아니다.

하드웨어 counter의 측정 범위는 플랫폼에 따라 다르다. Linux에서는 커널 실행을 제외하고
자식 프로세스를 포함한다. macOS의 cycles와 instructions는 커널 실행을 포함하고 자식 프로세스를 제외한다.

[확장 규약과 도구 조사](docs/metrics-implementation.md)에서 probe와 통계 저장 방식을 다룬다.
[hyperfine 지표 정의](https://github.com/sharkdp/hyperfine/blob/v2.0.0/README.md#choosing-metrics-and-units)에 플랫폼별 제약이 정리되어 있다.
