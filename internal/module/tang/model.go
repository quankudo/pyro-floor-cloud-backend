package tang

type Tang struct {
	IDTang   string
	IDToaNha string
	SoTang   int
}

type FloorsResponse struct {
	IDTang string `json:"id_tang"`
	SoTang int    `json:"so_tang"`
}
