export default {
    name: 'GroupImportParticipants',
    data() {
        return {
            loading: false,
            group: '',
            file: null,
            fileName: '',
            previewPhones: [],   // nomor HP yang berhasil di-preview sebelum submit
            previewError: '',
            results: [],         // hasil dari API setelah submit
        };
    },
    computed: {
        group_id() {
            const g = String(this.group ?? '').trim();
            if (!g) return '';
            return g.endsWith(window.TYPEGROUP) ? g : `${g}${window.TYPEGROUP}`;
        },
        isValidForm() {
            return String(this.group ?? '').trim() !== '' && this.file !== null;
        },
        successCount() {
            return this.results.filter(r => r.status === 'success').length;
        },
        errorCount() {
            return this.results.filter(r => r.status !== 'success').length;
        },
    },
    methods: {
        openModal() {
            this.handleReset();
            $('#modalGroupImportParticipants').modal({
                onApprove: () => false,
                closable: true,
            }).modal('show');
        },

        // ── File pick ────────────────────────────────────────────────────────
        onFileChange(event) {
            const f = event.target.files[0];
            if (!f) return;

            const allowed = ['.xlsx', '.xls', '.csv'];
            const ext = f.name.slice(f.name.lastIndexOf('.')).toLowerCase();
            if (!allowed.includes(ext)) {
                this.previewError = `Format tidak didukung. Gunakan ${allowed.join(', ')}`;
                this.file = null;
                this.fileName = '';
                this.previewPhones = [];
                return;
            }

            this.file = f;
            this.fileName = f.name;
            this.previewError = '';
            this.results = [];

            // Preview lokal: baca CSV langsung di browser
            if (ext === '.csv') {
                this.previewCSV(f);
            } else {
                // Untuk Excel, tampilkan pesan tanpa preview
                this.previewPhones = [];
            }
        },

        previewCSV(file) {
            const reader = new FileReader();
            reader.onload = (e) => {
                const text = e.target.result;
                const lines = text.split(/\r?\n/).filter(l => l.trim());
                const phones = [];
                for (const line of lines) {
                    const col = line.split(',')[0].replace(/[^0-9]/g, '').trim();
                    if (col) phones.push(col);
                }
                this.previewPhones = phones;
            };
            reader.readAsText(file);
        },

        triggerFilePicker() {
            this.$refs.fileInput.click();
        },

        // ── Submit ───────────────────────────────────────────────────────────
        async handleSubmit() {
            if (!this.isValidForm || this.loading) return;

            this.results = [];
            this.loading = true;

            try {
                const formData = new FormData();
                // Kirim group_id tanpa suffix — backend sudah handle via SanitizePhone
                formData.append('group_id', this.group_id);
                formData.append('file', this.file);

                const response = await window.http.post('/group/participants/import', formData, {
                    headers: { 'Content-Type': 'multipart/form-data' },
                });

                this.results = response.data.results ?? [];
                showSuccessInfo(response.data.message);
            } catch (err) {
                const msg = err.response?.data?.message ?? err.message;
                showErrorInfo(msg);
            } finally {
                this.loading = false;
            }
        },

        handleReset() {
            this.group = '';
            this.file = null;
            this.fileName = '';
            this.previewPhones = [];
            this.previewError = '';
            this.results = [];
            if (this.$refs.fileInput) this.$refs.fileInput.value = '';
        },

        downloadTemplate() {
            const header = 'phone_number\n';
            const example = '6281234567890\n6287654321098\n';
            const blob = new Blob([header + example], { type: 'text/csv' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'template_participants.csv';
            a.click();
            URL.revokeObjectURL(url);
        },
    },

    template: `
<div class="green card" @click="openModal" style="cursor: pointer">
    <div class="content">
        <a class="ui green right ribbon label">Group</a>
        <div class="header">Import Participants</div>
        <div class="description">
            Tambah anggota grup secara massal via Excel / CSV
        </div>
    </div>
</div>

<!-- Modal -->
<div class="ui small modal" id="modalGroupImportParticipants">
    <i class="close icon"></i>
    <div class="header">
        <i class="file excel icon"></i> Import Participants dari File
    </div>

    <div class="content">
        <form class="ui form">

            <!-- Group ID -->
            <div class="field">
                <label>Group ID</label>
                <input v-model="group" type="text"
                       placeholder="12036322888236XXXX (tanpa @g.us)"
                       aria-label="Group ID" />
                <div v-if="group_id" style="margin-top:4px">
                    <small class="ui grey text">
                        <i class="info circle icon"></i>
                        Akan dikirim ke: <code>[[ group_id ]]</code>
                    </small>
                </div>
            </div>

            <!-- File Upload -->
            <div class="field">
                <label>File (.xlsx, .xls, .csv)</label>

                <!-- Hidden native input -->
                <input ref="fileInput" type="file"
                       accept=".xlsx,.xls,.csv"
                       style="display:none"
                       @change="onFileChange" />

                <div style="display:flex; gap:8px; align-items:center; flex-wrap:wrap">
                    <button class="ui primary button" type="button" @click="triggerFilePicker">
                        <i class="upload icon"></i> Pilih File
                    </button>
                    <button class="ui basic button" type="button" @click="downloadTemplate"
                            title="Download template CSV">
                        <i class="download icon"></i> Template CSV
                    </button>
                    <span v-if="fileName" class="ui label">
                        <i class="file icon"></i> [[ fileName ]]
                    </span>
                </div>

                <!-- Error format -->
                <div v-if="previewError" class="ui red message" style="margin-top:8px">
                    <i class="exclamation triangle icon"></i> [[ previewError ]]
                </div>

                <!-- Preview CSV -->
                <div v-if="previewPhones.length > 0" style="margin-top:10px">
                    <div class="ui message" style="padding:10px">
                        <p>
                            <i class="eye icon"></i>
                            <strong>Preview:</strong> [[ previewPhones.length ]] nomor ditemukan
                        </p>
                        <div style="max-height:140px; overflow-y:auto; font-size:0.85em; font-family:monospace">
                            <div v-for="(p, i) in previewPhones.slice(0, 20)" :key="i">[[ i+1 ]]. [[ p ]]</div>
                            <div v-if="previewPhones.length > 20" style="color:grey">
                                … dan [[ previewPhones.length - 20 ]] nomor lainnya
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Info untuk Excel -->
                <div v-if="file && !fileName.endsWith('.csv')" style="margin-top:8px">
                    <div class="ui info message" style="padding:10px; font-size:0.88em">
                        <i class="info circle icon"></i>
                        File Excel dipilih. Preview tidak tersedia di browser —
                        nomor akan dibaca dari <strong>kolom pertama</strong> sheet pertama saat submit.
                    </div>
                </div>

                <div style="margin-top:8px; font-size:0.82em; color:#888">
                    Format: baris pertama boleh header, <strong>kolom pertama</strong> = nomor HP internasional (contoh: 6281234567890)
                </div>
            </div>

            <!-- Results -->
            <div v-if="results.length > 0" style="margin-top:10px">
                <div class="ui message" :class="errorCount > 0 ? 'warning' : 'success'">
                    <div class="header">
                        Hasil: [[ successCount ]] berhasil, [[ errorCount ]] gagal
                    </div>
                </div>
                <div style="max-height:160px; overflow-y:auto; margin-top:6px">
                    <table class="ui very compact small table">
                        <thead>
                            <tr>
                                <th>Nomor</th>
                                <th>Status</th>
                                <th>Keterangan</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="(r, i) in results" :key="i"
                                :class="r.status === 'success' ? '' : 'error'">
                                <td style="font-family:monospace; font-size:0.85em">[[ r.participant ]]</td>
                                <td>
                                    <span :class="r.status === 'success' ? 'ui green label' : 'ui red label'"
                                          style="font-size:0.78em">
                                        [[ r.status ]]
                                    </span>
                                </td>
                                <td style="font-size:0.85em">[[ r.message ]]</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

        </form>
    </div>

    <div class="actions">
        <button class="ui basic button" type="button" @click="handleReset">
            <i class="redo icon"></i> Reset
        </button>
        <button class="ui approve positive right labeled icon button" type="button"
                :class="{ loading: loading, disabled: !isValidForm || loading }"
                @click.prevent="handleSubmit">
            Import
            <i class="users icon"></i>
        </button>
    </div>
</div>
    `,
};
