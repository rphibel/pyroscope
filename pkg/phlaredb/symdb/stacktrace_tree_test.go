package symdb

import (
	"bytes"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/pyroscope/pkg/pprof"
)

func Test_stacktrace_tree_encoding(t *testing.T) {
	stacks := [][]uint64{
		{5, 4, 3, 2, 1},
		{6, 4, 3, 2, 1},
		{4, 3, 2, 1},
		{3, 2, 1},
		{4, 2, 1},
		{7, 2, 1},
		{2, 1},
		{1},
	}

	x := newStacktraceTree(10)
	var b bytes.Buffer

	for i := range stacks {
		x.insert(stacks[i])

		b.Reset()
		_, err := x.WriteTo(&b)
		require.NoError(t, err)

		ppt := newParentPointerTree(x.len())
		_, err = ppt.ReadFrom(bytes.NewBuffer(b.Bytes()))
		require.NoError(t, err)

		for j := range x.nodes {
			n, p := x.nodes[j], ppt.nodes[j]
			if n.p != p.p || n.r != p.r {
				t.Fatalf("tree mismatch on %v: n:%#v, p:%#v", stacks[i], n, p)
			}
		}
	}
}

func Test_stacktrace_tree_encoding_group(t *testing.T) {
	stacks := [][]uint64{
		{5, 4, 3, 2, 1},
		{6, 4, 3, 2, 1},
		{4, 3, 2, 1},
		{3, 2, 1},
		{4, 2, 1},
		{7, 2, 1},
		{2, 1},
		{1},
	}

	x := newStacktraceTree(10)
	var b bytes.Buffer

	for i := range stacks {
		x.insert(stacks[i])

		b.Reset()
		e := treeEncoder{writeSize: 30}
		err := e.marshal(x, &b)
		require.NoError(t, err)

		ppt := newParentPointerTree(x.len())
		d := treeDecoder{
			bufSize:     64,
			peekSize:    20,
			groupBuffer: 12,
		}
		err = d.unmarshal(ppt, bytes.NewBuffer(b.Bytes()))
		require.NoError(t, err)

		for j := range x.nodes {
			n, p := x.nodes[j], ppt.nodes[j]
			if n.p != p.p || n.r != p.r {
				t.Fatalf("tree mismatch on %v: n:%#v, p:%#v", stacks[i], n, p)
			}
		}
	}
}

func Test_stacktrace_tree_encoding_rand(t *testing.T) {
	nodes := make([]node, 1<<20)
	for i := range nodes {
		nodes[i] = node{
			fc: 2,
			ns: 3,
			p:  int32(rand.Intn(10 << 10)),
			r:  int32(rand.Intn(10 << 10)),
		}
	}

	x := &stacktraceTree{nodes: nodes}
	var b bytes.Buffer
	_, err := x.WriteTo(&b)
	require.NoError(t, err)

	ppt := newParentPointerTree(x.len())
	_, err = ppt.ReadFrom(bytes.NewBuffer(b.Bytes()))
	require.NoError(t, err)

	for j := range x.nodes {
		n, p := x.nodes[j], ppt.nodes[j]
		if n.p != p.p || n.r != p.r {
			t.Fatalf("tree mismatch at %d: n:%#v. p:%#v", j, n, p)
		}
	}
}

func Test_stacktrace_tree_pprof_locations_(t *testing.T) {
	x := newStacktraceTree(0)
	assert.Len(t, x.resolve([]int32{0, 1, 2, 3}, 42), 0)
	assert.Len(t, x.resolveUint64([]uint64{0, 1, 2, 3}, 42), 0)

	p := newParentPointerTree(0)
	assert.Len(t, p.resolve([]int32{0, 1, 2, 3}, 42), 0)
	assert.Len(t, p.resolveUint64([]uint64{0, 1, 2, 3}, 42), 0)
}

func Test_stacktrace_tree_pprof_locations(t *testing.T) {
	p, err := pprof.OpenFile("testdata/profile.pb.gz")
	require.NoError(t, err)

	x := newStacktraceTree(defaultStacktraceTreeSize)
	m := make(map[uint32]int)
	for i := range p.Sample {
		m[x.insert(p.Sample[i].LocationId)] = i
	}

	tmp := stacktraceLocations.get()
	defer stacktraceLocations.put(tmp)
	for sid, i := range m {
		tmp = x.resolve(tmp, sid)
		locs := p.Sample[i].LocationId
		for j := range locs {
			if tmp[j] != int32(locs[j]) {
				t.Log("resolved:", tmp)
				t.Log("locations:", locs)
				t.Fatalf("ST: tmp[j] != locs[j]")
			}
		}
	}

	var b bytes.Buffer
	n, err := x.WriteTo(&b)
	require.NoError(t, err)
	assert.Equal(t, b.Len(), int(n))

	ppt := newParentPointerTree(x.len())
	n, err = ppt.ReadFrom(bytes.NewReader(b.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, b.Len(), int(n))

	tmp = stacktraceLocations.get()
	defer stacktraceLocations.put(tmp)
	for sid, i := range m {
		tmp = ppt.resolve(tmp, sid)
		locs := p.Sample[i].LocationId
		for j := range locs {
			if tmp[j] != int32(locs[j]) {
				t.Log("resolved:", tmp)
				t.Log("locations:", locs)
				t.Fatalf("PPT: tmp[j] != locs[j]")
			}
		}
	}
}

// The test is helpful for debugging.
func Test_parentPointerTree_toStacktraceTree(t *testing.T) {
	x := newStacktraceTree(10)
	for _, stack := range [][]uint64{
		{5, 4, 3, 2, 1},
		{6, 4, 3, 2, 1},
		{4, 3, 2, 1},
		{3, 2, 1},
		{4, 2, 1},
		{7, 2, 1},
		{2, 1},
		{1},
	} {
		x.insert(stack)
	}
	assertRestoredStacktraceTree(t, x)
}

func Test_parentPointerTree_toStacktraceTree_profile(t *testing.T) {
	p, err := pprof.OpenFile("testdata/profile.pb.gz")
	require.NoError(t, err)
	x := newStacktraceTree(defaultStacktraceTreeSize)
	for _, s := range p.Sample {
		x.insert(s.LocationId)
	}
	assertRestoredStacktraceTree(t, x)
}

func assertRestoredStacktraceTree(t *testing.T, x *stacktraceTree) {
	var b bytes.Buffer
	_, _ = x.WriteTo(&b)
	ppt := newParentPointerTree(x.len())
	_, err := ppt.ReadFrom(bytes.NewBuffer(b.Bytes()))
	require.NoError(t, err)
	restored := ppt.toStacktraceTree()
	assert.Equal(t, x.nodes, restored.nodes)
}

func Benchmark_stacktrace_tree_insert(b *testing.B) {
	p, err := pprof.OpenFile("testdata/profile.pb.gz")
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		x := newStacktraceTree(defaultStacktraceTreeSize)
		for j := range p.Sample {
			x.insert(p.Sample[j].LocationId)
		}
	}
}

func Benchmark_stacktrace_tree_insert_default_sizes(b *testing.B) {
	p, err := pprof.OpenFile("testdata/profile.pb.gz")
	require.NoError(b, err)

	b.ResetTimer()

	for _, size := range []int{0, 10, 1024, 2048, 4096, 8192} {
		b.Run("size="+strconv.Itoa(size), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				x := newStacktraceTree(size)
				for j := range p.Sample {
					x.insert(p.Sample[j].LocationId)
				}

				if testing.Verbose() {
					c := float64(cap(x.nodes))
					b.ReportMetric(c, "cap")
					b.ReportMetric(c*float64(stacktraceTreeNodeSize), "size")
					b.ReportMetric(float64(x.len())/float64(c)*100, "fill")
				}
			}
		})
	}
}
