<template>
  <div ref="page" tabindex="-1" class="p-page p-page-labels not-selectable">
    <v-form ref="form" validate-on="invalid-input" class="p-labels-search p-page__navigation" @submit.stop.prevent="fetchWorlds()">
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
          @keyup.enter="fetchWorlds()"
          @click:clear="clearSearch()"
        ></v-text-field>
      </v-toolbar>
    </v-form>

    <div v-if="loading" class="p-page__loading">
      <p-loading></p-loading>
    </div>
    <div v-else class="p-page__content">
      <div v-if="results.length === 0" class="pa-3">
        <v-alert color="surface-variant" icon="mdi-earth" class="no-results" variant="outlined">
          <div class="font-weight-bold">
            {{ $gettext("No worlds found") }}
          </div>
          <div class="mt-2">
            {{ $gettext("No VRChat world metadata was found in your indexed photos.") }}
          </div>
        </v-alert>
      </div>
      <div v-else class="v-row search-results label-results cards-view">
        <div v-for="world in results" :key="world.Name" class="v-col-6 v-col-sm-4 v-col-md-3 v-col-xl-2">
          <div class="result not-selectable clickable" @click="openWorld(world)">
            <div :title="world.Name" :style="`background-image: url(${thumbnailUrl(world)})`" class="preview">
              <div class="preview__overlay"></div>
            </div>

            <div class="meta">
              <div class="meta-title">
                {{ world.Name }}
              </div>
              <div v-if="world.PhotoCount === 1" class="meta-count">
                {{ $gettext("Contains one picture.") }}
              </div>
              <div v-else-if="world.PhotoCount > 0" class="meta-count">
                {{ $gettext("Contains %{n} pictures.", { n: world.PhotoCount }) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import $api from "common/api";
import { $config } from "app/session";
import PLoading from "component/loading.vue";

export default {
  name: "PPageWorlds",
  components: {
    PLoading,
  },
  data() {
    return {
      loading: true,
      query: "",
      results: [],
    };
  },
  computed: {
    density() {
      return this.$vuetify.display.smAndDown ? "compact" : "comfortable";
    },
  },
  created() {
    this.fetchWorlds();
  },
  methods: {
    fetchWorlds() {
      this.loading = true;
      const params = { count: 240, offset: 0 };
      if (this.query) {
        params.q = this.query;
      }

      $api
        .get("worlds", { params })
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
      this.fetchWorlds();
    },
    thumbnailUrl(world) {
      if (world && world.Thumb) {
        return `${$config.contentUri}/t/${world.Thumb}/${$config.previewToken}/tile_500`;
      }
      return `${$config.contentUri}/svg/label`;
    },
    openWorld(world) {
      const target = world.WorldID || world.Name || "";
      const escaped = String(target).replace(/"/g, '\\"');
      this.$router.push({
        name: "photos",
        query: { q: `world:"${escaped}"` },
      });
    },
  },
};
</script>
