
# Config

One can use config files that are TOML, YAML, or JSON based, just include `.json|.js`, `.yml|.yaml` or `.toml` extenstions to the files in question

I'll show configs using the TOML format, because i think it's easier to read then yaml for complex items.

Here we'll attempt to go through the 4 main config files, "config.toml", "prereg.toml", "injector.toml", and "api.toml".

## run modes

### Consistent Hash mode

    cadent -config=config.toml

### Consistent Hash mode + Writer Mode

     cadent -config=config.toml -prereg=prereg.toml

### Injector mode (aka kafka stream)

    cadent -injector=injector.toml
    
### API only mode (alpha)

    cadent -api=api.toml

## tomlenv

We use a modified toml format that accepts environment variables as well.

The general format for using the ENV is

    $ENV{MY_ENV_VAR:defaultvalue}

at the moment "multiline" default values are not supported (and not really needed here, yet)


##  [config.toml](./configtoml.md)

##  [prereg.toml](./preregtoml.md)

##  [api.toml](./apitoml.md)

##  [injector.toml](./injectortoml.md)
