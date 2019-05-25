package modelgen

import (
	"context"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"

	g "github.com/sunfmin/gogen"
)

type ModelsCodeBuilder struct {
	mb  *ModelBuild
	cfg *config.Config
	fh  *templates.FuncsHelper
}

func ModelsCode(mbuild *ModelBuild, cfg *config.Config) (r *ModelsCodeBuilder) {
	r = &ModelsCodeBuilder{
		mb:  mbuild,
		cfg: cfg,
	}
	return
}

func (b *ModelsCodeBuilder) SetFuncsHelper(fh *templates.FuncsHelper) {
	b.fh = fh
	return
}

func (b *ModelsCodeBuilder) MarshalCode(ctx context.Context) (r []byte, err error) {
	fh := b.fh
	fh.ReserveImport("context")
	fh.ReserveImport("fmt")
	fh.ReserveImport("io")
	fh.ReserveImport("strconv")
	fh.ReserveImport("time")
	fh.ReserveImport("sync")
	fh.ReserveImport("errors")
	fh.ReserveImport("bytes")
	fh.ReserveImport("github.com/vektah/gqlparser")
	fh.ReserveImport("github.com/vektah/gqlparser/ast")
	fh.ReserveImport("github.com/99designs/gqlgen/graphql")
	fh.ReserveImport("github.com/99designs/gqlgen/graphql/introspection")

	root := g.Codes()

	for _, m := range b.mb.Interfaces {
		root.Append(
			g.LineComment(m.Description),
			g.Block(`
				type $name interface {
					Is$name()
				}
			`, "$name", fh.Go(m.Name)),
		)
	}

	root.Append(g.RawCode("\n"))
	for _, m := range b.mb.Models {
		s := g.Struct(fh.Go(m.Name))

		for _, f := range m.Fields {
			s.AppendFieldComment(f.Description)
			s.AppendField(fh.Go(f.Name), fh.Ref(f.Type), f.Tag)
		}

		imps := g.Codes()
		for _, im := range m.Implements {
			imps.Append(g.Block("func ($Rec) Is$Im() {}", "$Rec", fh.Go(m.Name), "$Im", fh.Go(im)))
		}
		root.Append(
			g.LineComment(m.Description),
			s,
			imps,
		)
	}

	for _, e := range b.mb.Enums {
		typ := fh.Go(e.Name)
		constVals := g.Codes().Separator(",\n", true)
		cst := g.ConstBlock().Type(typ, "string")
		for _, v := range e.Values {
			name := fh.Go(v.Name)
			cst.Consts(g.LineComment(v.Description))
			cst.AppendConst(name, v.Name)

			constVals.Append(g.Block("$Type$Val", "$Type", typ, "$Val", name))
		}

		root.Append(
			g.LineComment(e.Description),
			cst,
			g.Block(`var All$Type = []$Type {
					$Vals
				}`, "$Type", typ).
				VarBlock("$Vals", constVals),
			g.Block(`func (e $Type) IsValid() bool {
				switch e {
				case $Vals:
					return true
				}
				return false
			}`, "$Type", typ).
				VarBlock("$Vals", constVals.Clone().Separator(", ", false)),
			g.Block(`
func (e $Type) String() string {
	return string(e)
}

func (e *$Type) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = $Type(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid $Name", str)
	}
	return nil
}

func (e $Type) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

			`, "$Type", typ, "$Name", e.Name),
		)
	}

	return root.MarshalCode(ctx)

}
