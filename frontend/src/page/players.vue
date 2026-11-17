<template>
  <div ref="page" tabindex="-1" class="p-page p-page-labels not-selectable">
    <v-form ref="form" validate-on="invalid-input" class="p-labels-search p-page__navigation" @submit.stop.prevent="fetchPlayers()">
      <v-toolbar flat :density="$vuetify.display.smAndDown ? 'compact' : 'default'" color="secondary" class="page-toolbar">
        <v-text-field
          v-model.trim="query"
          hide-details
          clearable
          overflow
          single-line
          rounded
          variant="solo-filled"
          :density="density"
          validate-on="invalid-input"
          :placeholder="$gettext('Search')"
          prepend-inner-icon="mdi-magnify"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="none"
          color="surface-variant"
          class="input-search input-search--focus background-inherit elevation-0"
          @keyup.enter="fetchPlayers()"
          @click:clear="clearSearch()"
        ></v-text-field>

        <v-btn
          v-if="!query"
          variant="text"
          class="ms-2"
          @click.stop="toggleShowAll()"
        >
          {{ showAll ? $gettext("2+ Photos") : $gettext("Show All") }}
        </v-btn>
      </v-toolbar>
    </v-form>

    <div v-if="loading" class="p-page__loading">
      <p-loading></p-loading>
    </div>
    <div v-else class="p-page__content">
      <div v-if="results.length === 0" class="pa-3">
        <v-alert color="surface-variant" icon="mdi-account-group" class="no-results" variant="outlined">
          <div class="font-weight-bold">
            {{ $gettext("No players found") }}
          </div>
          <div class="mt-2">
            {{ $gettext("No VRChat player metadata matched your filter.") }}
          </div>
        </v-alert>
      </div>
      <div v-else class="v-row search-results label-results cards-view">
        <div v-for="player in results" :key="player.Name" class="v-col-6 v-col-sm-4 v-col-md-3 v-col-xl-2">
          <div class="result not-selectable clickable" @click="openPlayer(player)">
            <div :title="player.Name" :style="`background-image: url(${thumbnailUrl(player)})`" class="preview">
              <div class="preview__overlay"></div>
            </div>

            <div class="meta">
              <div class="meta-title">
                {{ player.Name }}
              </div>
              <div v-if="player.PhotoCount === 1" class="meta-count">
                {{ $gettext("Contains one picture.") }}
              </div>
              <div v-else-if="player.PhotoCount > 0" class="meta-count">
                {{ $gettext("Contains %{n} pictures.", { n: player.PhotoCount }) }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="results.length && !showAll && !query" class="d-flex justify-center my-8">
        <v-btn color="button" rounded variant="flat" @click.stop="toggleShowAll()">
          {{ $gettext("Show All Players") }}
        </v-btn>
      </div>
    </div>
  </div>
</template>

<script>
import $api from "common/api";
import { $config } from "app/session";
import PLoading from "component/loading.vue";

export default {
  name: "PPagePlayers",
  components: {
    PLoading,
  },
  data() {
    return {
      loading: true,
      query: "",
      showAll: false,
      results: [],
    };
  },
  computed: {
    density() {
      return this.$vuetify.display.smAndDown ? "compact" : "comfortable";
    },
  },
  created() {
    this.fetchPlayers();
  },
  methods: {
    fetchPlayers() {
      this.loading = true;
      const params = { count: 240, offset: 0 };
      if (this.query) {
        params.q = this.query;
      }
      if (this.showAll) {
        params.all = true;
      }

      $api
        .get("players", { params })
        .then((response) => {
          this.results = Array.isArray(response.data) ? response.data : [];
        })
        .catch(() => {
          this.results = [];
        })
        .finally(() => {
          this.loading = false;
        });
    },
    clearSearch() {
      this.query = "";
      this.fetchPlayers();
    },
    toggleShowAll() {
      this.showAll = !this.showAll;
      this.fetchPlayers();
    },
    thumbnailUrl(player) {
      if (player && player.Thumb) {
        return `${$config.contentUri}/t/${player.Thumb}/${$config.previewToken}/tile_500`;
      }
      return `${$config.contentUri}/svg/label`;
    },
    openPlayer(player) {
      const target = player.PlayerID || player.Name || "";
      const escaped = String(target).replace(/"/g, '\\"');
      this.$router.push({
        name: "photos",
        query: { q: `player:"${escaped}"` },
      });
    },
  },
};
</script>
